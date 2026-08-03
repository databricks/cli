package aircmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// airRunWithCompute builds a run reporting the given accelerator type and count.
func airRunWithCompute(accelType string, count int) *jobs.Run {
	return &jobs.Run{
		RunId: 123,
		Tasks: []jobs.RunTask{{
			RunId: 456,
			AiRuntimeTask: &jobs.AiRuntimeTask{
				Deployments: []jobs.DeploymentSpec{{
					Compute: jobs.ComputeSpec{
						AcceleratorType:  jobs.ComputeSpecAcceleratorType(accelType),
						AcceleratorCount: count,
					},
				}},
			},
		}},
	}
}

func TestResolveNodeCount(t *testing.T) {
	tests := []struct {
		accelType string
		count     int
		want      int
	}{
		{"GPU_1xA10", 2, 2},
		{"GPU_1xH100", 4, 4},
		{"GPU_8xH100", 16, 2},
	}
	for _, tt := range tests {
		n, err := resolveNodeCount(airRunWithCompute(tt.accelType, tt.count))
		require.NoError(t, err)
		assert.Equal(t, tt.want, n)
	}

	// A run with no AI runtime compute errors.
	_, err := resolveNodeCount(&jobs.Run{RunId: 1})
	require.Error(t, err)

	// A count that isn't a whole number of nodes can't be mapped to node indices,
	// so it errors rather than truncating to 0.
	_, err = resolveNodeCount(airRunWithCompute("GPU_8xH100", 4))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a multiple of 8")

	// A zero count is reported as such, not as a missing config.
	_, err = resolveNodeCount(airRunWithCompute("GPU_1xA10", 0))
	require.Error(t, err)
}

// downloadServer serves the MLflow artifact chain: the artifact listing, a
// pre-signed URL pointing back at itself, and the chunk bytes.
func downloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/list":
			if r.URL.Query().Get("path") == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}, {"path": "logs/node_1", "is_dir": true}]}`))
			} else {
				_, _ = w.Write([]byte(`{"files": [{"path": "` + r.URL.Query().Get("path") + `/logs-0.chunk.txt", "file_size": 12}]}`))
			}
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte("line one\nline two\n"))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadNodeLogWritesConcatenatedChunks(t *testing.T) {
	w := newTestWorkspaceClient(t, downloadServer(t).URL)
	dir := t.TempDir()

	path, err := downloadNodeLog(t.Context(), w, "run1", 0, 0, false, dir)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\n", string(got))
	assert.Equal(t, filepath.Join(dir, "logs", "node_0.log"), path)
}

func TestDownloadAllNodeLogs(t *testing.T) {
	w := newTestWorkspaceClient(t, downloadServer(t).URL)
	dir := t.TempDir()

	nodeLogs, failures, err := downloadAllNodeLogs(t.Context(), w, "run1", dir, []int{0, 1}, -1)
	require.NoError(t, err)
	require.Empty(t, failures)
	require.Len(t, nodeLogs, 2)
	assert.FileExists(t, nodeLogs[0])
	assert.FileExists(t, nodeLogs[1])
}

// fullDownloadServer also serves the run and its output, so downloadLogs can run
// end to end against a 2-node run.
func fullDownloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	runGet := `{
		"run_id": 123,
		"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
		"tasks": [{"run_id": 456, "ai_runtime_task": {"deployments": [{"compute": {"accelerator_type": "GPU_1xA10", "accelerator_count": 2}}]}}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runGet))
		case "/api/2.2/jobs/runs/get-output":
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`))
		case "/api/2.0/mlflow/artifacts/list":
			if r.URL.Query().Get("path") == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}, {"path": "logs/node_1", "is_dir": true}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"files": [{"path": "` + r.URL.Query().Get("path") + `/logs-0.chunk.txt", "file_size": 6}]}`))
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte("hello\n"))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadLogsAllNodes(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	success, err := downloadLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 123, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.True(t, success)
	assert.FileExists(t, filepath.Join(dir, "logs", "node_0.log"))
	assert.FileExists(t, filepath.Join(dir, "logs", "node_1.log"))
}

func TestDownloadLogsSingleNode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	success, err := downloadLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 123, node: 1, nodeSet: true, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.True(t, success)
	assert.FileExists(t, filepath.Join(dir, "logs", "node_1.log"))
	assert.NoFileExists(t, filepath.Join(dir, "logs", "node_0.log"))
}

func TestDownloadLogsOutOfRangeNode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	_, err := downloadLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 123, node: 5, nodeSet: true, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --node 5")
}

func TestDownloadLogsExplicitNodeZero(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	// An explicit --node 0 must download only node 0, unlike the default which
	// downloads every node.
	_, err := downloadLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 123, node: 0, nodeSet: true, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "logs", "node_0.log"))
	assert.NoFileExists(t, filepath.Join(dir, "logs", "node_1.log"))
}

func TestDownloadLogsOutOfRangeNodeIsInvalidArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)

	// The sentinel lets the caller classify this as bad input rather than a
	// transient, retryable failure.
	_, err := downloadLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 123, node: 5, nodeSet: true, attempt: -1, downloadTo: t.TempDir()}, logRunStatus{resultState: "SUCCESS"})
	require.ErrorIs(t, err, errNodeOutOfRange)
}

// noLogsDownloadServer serves a run with no resolvable MLflow run, so there is
// nothing to download.
func noLogsDownloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{
				"run_id": 123,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"run_id": 456, "ai_runtime_task": {"deployments": [{"compute": {"accelerator_type": "GPU_1xA10", "accelerator_count": 2}}]}}]
			}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadLogsNoLogsMatchesStreamingPath(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, noLogsDownloadServer(t).URL)

	// A SUCCESS run with no logs reports it and still succeeds, exactly as the
	// streaming path does — no error, no non-zero exit.
	var buf bytes.Buffer
	success, err := downloadLogs(ctx, w, &buf, logRequest{runID: 123, attempt: -1, downloadTo: t.TempDir()}, logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.True(t, success)
	assert.Contains(t, buf.String(), "No logs available for run 123")
}

// partialFailureServer fails node 1's chunk listing so one node succeeds and the
// other doesn't.
func partialFailureServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/list":
			p := r.URL.Query().Get("path")
			switch p {
			case "logs":
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}, {"path": "logs/node_1", "is_dir": true}]}`))
			case "logs/node_1":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_code": "INTERNAL", "message": "boom"}`))
			default:
				_, _ = w.Write([]byte(`{"files": [{"path": "` + p + `/logs-0.chunk.txt"}]}`))
			}
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte("ok\n"))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadAllNodeLogsReportsPartialFailure(t *testing.T) {
	w := newTestWorkspaceClient(t, partialFailureServer(t).URL)
	dir := t.TempDir()

	// Node 1 fails, but node 0 still downloads and the failure is reported rather
	// than silently dropped.
	nodeLogs, failures, err := downloadAllNodeLogs(t.Context(), w, "run1", dir, []int{0, 1}, -1)
	require.NoError(t, err)
	require.Len(t, nodeLogs, 1)
	assert.FileExists(t, nodeLogs[0])
	require.Contains(t, failures, 1)
	assert.NotEmpty(t, failures[1])
}

func TestDownloadAllNodeLogsPropagatesCancellation(t *testing.T) {
	w := newTestWorkspaceClient(t, downloadServer(t).URL)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// A cancelled command must surface the cancellation, not look like a run with
	// no logs.
	_, _, err := downloadAllNodeLogs(ctx, w, "run1", t.TempDir(), []int{0, 1}, -1)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// attemptPrefixServer serves the attempt-prefixed layout (logs/attempt_N/node_M).
func attemptPrefixServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/list":
			p := r.URL.Query().Get("path")
			if p == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/attempt_0", "is_dir": true}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"files": [{"path": "` + p + `/logs-0.chunk.txt"}]}`))
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			// Echo the path so the test can prove the attempt-prefixed dir was used.
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned?p=` + r.URL.Query().Get("path") + `"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte(r.URL.Query().Get("p") + "\n"))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadAllNodeLogsAttemptPrefixedLayout(t *testing.T) {
	w := newTestWorkspaceClient(t, attemptPrefixServer(t).URL)
	dir := t.TempDir()

	nodeLogs, _, err := downloadAllNodeLogs(t.Context(), w, "run1", dir, []int{0}, -1)
	require.NoError(t, err)
	require.Len(t, nodeLogs, 1)

	body, err := os.ReadFile(nodeLogs[0])
	require.NoError(t, err)
	assert.Contains(t, string(body), "logs/attempt_0/node_0")
}
