package aircmd

import (
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

	path, err := downloadNodeLog(t.Context(), w, "run1", 0, 0, false, dir, 0)
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

	nodeLogs, err := downloadAllNodeLogs(t.Context(), w, "run1", dir, []int{0, 1}, -1, 0)
	require.NoError(t, err)
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

	success, err := downloadLogs(ctx, w, logRequest{runID: 123, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"}, dir)
	require.NoError(t, err)
	assert.True(t, success)
	assert.FileExists(t, filepath.Join(dir, "logs", "node_0.log"))
	assert.FileExists(t, filepath.Join(dir, "logs", "node_1.log"))
}

func TestDownloadLogsSingleNode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	success, err := downloadLogs(ctx, w, logRequest{runID: 123, node: 1, nodeSet: true, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"}, dir)
	require.NoError(t, err)
	assert.True(t, success)
	assert.FileExists(t, filepath.Join(dir, "logs", "node_1.log"))
	assert.NoFileExists(t, filepath.Join(dir, "logs", "node_0.log"))
}

func TestDownloadLogsOutOfRangeNode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)
	dir := t.TempDir()

	_, err := downloadLogs(ctx, w, logRequest{runID: 123, node: 5, nodeSet: true, attempt: -1, downloadTo: dir}, logRunStatus{resultState: "SUCCESS"}, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node 5 does not exist")
}

// multiChunkServer serves a node dir with three chunks whose bodies identify
// which chunk was fetched, so a chunk limit is observable.
func multiChunkServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/list":
			_, _ = w.Write([]byte(`{"files": [
				{"path": "logs/node_0/logs-0.chunk.txt"},
				{"path": "logs/node_0/logs-1.chunk.txt"},
				{"path": "logs/node_0/logs-2.chunk.txt"}
			]}`))
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			// Echo the requested artifact path back through the pre-signed URL so
			// the downloaded body identifies its chunk.
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

func TestDownloadNodeLogLimitsChunksFromEnd(t *testing.T) {
	w := newTestWorkspaceClient(t, multiChunkServer(t).URL)

	// No limit: all three chunks are concatenated.
	full, err := downloadNodeLog(t.Context(), w, "run1", 0, 0, false, t.TempDir(), 0)
	require.NoError(t, err)
	body, err := os.ReadFile(full)
	require.NoError(t, err)
	assert.Contains(t, string(body), "logs-0.chunk.txt")
	assert.Contains(t, string(body), "logs-2.chunk.txt")

	// Limit of 2 keeps only the two newest chunks.
	limited, err := downloadNodeLog(t.Context(), w, "run1", 0, 0, false, t.TempDir(), 2)
	require.NoError(t, err)
	body, err = os.ReadFile(limited)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "logs-0.chunk.txt")
	assert.Contains(t, string(body), "logs-1.chunk.txt")
	assert.Contains(t, string(body), "logs-2.chunk.txt")
}
