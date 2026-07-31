package aircmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCommandShape(t *testing.T) {
	cmd := newLogsCommand()
	assert.Equal(t, "logs JOB_RUN_ID", cmd.Use)
	assert.Empty(t, cmd.Commands(), "logs must not register subcommands")
	assert.NoError(t, cmd.Args(cmd, []string{"123"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"1", "2"}))

	// --review is hidden.
	review := cmd.Flags().Lookup("review")
	require.NotNil(t, review)
	assert.True(t, review.Hidden)
}

// runLogsCmd invokes the logs command's RunE with the given flags against a mock
// (no-HTTP) workspace client. Used for input validation that fails before any
// API call.
func runLogsCmd(t *testing.T, args []string, flagsToSet map[string]string) error {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newLogsCommand(), flags.OutputText)
	for k, v := range flagsToSet {
		require.NoError(t, cmd.Flags().Set(k, v))
	}
	cmd.SetContext(ctx)
	return cmd.RunE(cmd, args)
}

func TestLogsFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantMsg string
	}{
		{
			name:    "lines and minutes are mutually exclusive",
			args:    []string{"5"},
			flags:   map[string]string{"lines": "100", "minutes": "10"},
			wantMsg: "cannot combine --lines with --minutes",
		},
		{
			name:    "negative lines rejected",
			args:    []string{"5"},
			flags:   map[string]string{"lines": "-1"},
			wantMsg: "invalid --lines",
		},
		{
			name:    "negative minutes rejected",
			args:    []string{"5"},
			flags:   map[string]string{"minutes": "-1"},
			wantMsg: "invalid --minutes",
		},
		{
			name:    "review not implemented",
			args:    []string{"5"},
			flags:   map[string]string{"review": "true"},
			wantMsg: "--review is not implemented yet",
		},
		{
			name:    "invalid run id",
			args:    []string{"abc"},
			wantMsg: "invalid JOB_RUN_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runLogsCmd(t, tt.args, tt.flags)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

// completedRunLogsServer serves the auth probe, a terminal runs/get, and a
// single page of Bricklens logs (newest-first, as the tail fetch requests).
func completedRunLogsServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{
				"run_id": 5,
				"start_time": 1000,
				"end_time": 2000,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"attempt_number": 0}]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			// Newest-first, as drainTail requests; reversed to oldest-first on print.
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 2000000000, "body": "line two", "node_index": 0},
				{"time_unix_nano": 1000000000, "body": "line one", "node_index": 0}
			]}`))
		default:
			// Me() probe and SDK config discovery.
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsCompletedRunTail(t *testing.T) {
	srv := completedRunLogsServer(t)
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	// Drive runLogs directly (bypassing PreRunE auth wiring) with a resolved request.
	err := runLogs(ctx, cmd, logRequest{runID: 5, node: 0, attempt: -1, tailLines: -1})
	require.NoError(t, err)

	// Records print oldest-first regardless of the newest-first fetch order.
	assert.Equal(t, "line one\nline two\n", buf.String())
}

// mlflowFallbackServer serves a terminal run whose Bricklens endpoint is gated
// off (FEATURE_DISABLED), plus the full MLflow artifact path the fallback walks:
// runs/get-output (MLflow ids), artifacts/list (logs dir + chunk file),
// credentials-for-read (pre-signed URL), and the pre-signed chunk bytes itself.
func mlflowFallbackServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{
				"run_id": 5,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"run_id": 456, "attempt_number": 0}]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code": "FEATURE_DISABLED", "message": "bricklens logs gated off"}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`))
		case r.URL.Path == "/api/2.0/mlflow/artifacts/list":
			// The logs dir probe (format discovery) and the per-node chunk listing
			// both hit this; return the old-format node dir and one chunk file.
			if r.URL.Query().Get("path") == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0/logs-0.chunk.txt", "file_size": 12}]}`))
		case r.URL.Path == "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case r.URL.Path == "/presigned":
			_, _ = w.Write([]byte("line one\nline two\n"))
		default:
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsFallsBackToMLflow(t *testing.T) {
	srv := mlflowFallbackServer(t)
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	// Bricklens is gated off, so fetchLogs routes to the MLflow fallback, which
	// resolves the MLflow run, lists the chunk, downloads it via the pre-signed
	// URL, and prints its lines.
	err := runLogs(ctx, cmd, logRequest{runID: 5, node: 0, attempt: -1, tailLines: -1})
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\n", buf.String())
}

// activeRunPastRetryServer serves a still-RUNNING run with two attempts and a
// single page of Bricklens logs. runs/get always returns RUNNING; a test that
// follows the run would poll forever, so it also asserts the static path never
// loops.
func activeRunPastRetryServer(t *testing.T, getRunHits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			*getRunHits++
			_, _ = w.Write([]byte(`{
				"run_id": 9,
				"start_time": 1700000000000,
				"state": {"life_cycle_state": "RUNNING"},
				"tasks": [{"attempt_number": 0}, {"attempt_number": 1}]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 1700000001000000000, "body": "retry 0 log", "node_index": 0}
			]}`))
		default:
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsPastRetryOfActiveRunIsStatic(t *testing.T) {
	var getRunHits int
	srv := activeRunPastRetryServer(t, &getRunHits)
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	// --retry 0 on a RUNNING run whose latest attempt is 1: the past attempt's
	// logs render once instead of following the run (which would never terminate).
	// The run has no SUCCESS result yet, so it still exits non-zero via
	// ErrAlreadyPrinted; the logs are printed regardless.
	err := runLogs(ctx, cmd, logRequest{runID: 9, node: 0, attempt: 0, tailLines: -1})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, "retry 0 log\n", buf.String())

	// Exactly one runs/get: the initial status resolve in runLogs. The static
	// streamer must not re-poll (that is the loop this test guards against).
	assert.Equal(t, 1, getRunHits)
}
