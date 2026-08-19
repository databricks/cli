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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// watchServer serves submit, the auth probe, a terminal runs/get, and a single
// page of Bricklens logs — everything `air run --watch` touches after submit.
// resultState is the terminal result the run reports (e.g. SUCCESS, FAILED).
func watchServer(t *testing.T, resultState string) *httptest.Server {
	t.Helper()
	runGet := `{
		"run_id": 777,
		"start_time": 1700000000000,
		"end_time": 1700000012000,
		"state": {"life_cycle_state": "TERMINATED", "result_state": "` + resultState + `"},
		"tasks": [{"run_id": 778, "attempt_number": 0}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			_, _ = w.Write([]byte(`{"run_id": 777}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runGet))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 1700000002000000000, "body": "step 2", "node_index": 0},
				{"time_unix_nano": 1700000001000000000, "body": "step 1", "node_index": 0}
			]}`))
		default:
			// Me() probe, workspace-id, SDK config discovery.
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// watchServerMLflow serves a run whose Bricklens endpoint is gated off
// (FEATURE_DISABLED), forcing the MLflow fallback, plus the MLflow artifact
// chain (get-output, artifacts/list, credentials-for-read, the pre-signed bytes).
// STATUS events never fire on this path, so it guards the closing terminal
// envelope against relying on onStatusChange.
func watchServerMLflow(t *testing.T, resultState string) *httptest.Server {
	t.Helper()
	var base string
	runGet := `{
		"run_id": 777,
		"state": {"life_cycle_state": "TERMINATED", "result_state": "` + resultState + `"},
		"tasks": [{"run_id": 778, "attempt_number": 0}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			_, _ = w.Write([]byte(`{"run_id": 777}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runGet))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code": "FEATURE_DISABLED", "message": "gated off"}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`))
		case r.URL.Path == "/api/2.0/mlflow/artifacts/list":
			if r.URL.Query().Get("path") == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0/logs-0.chunk.txt", "file_size": 12}]}`))
		case r.URL.Path == "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case r.URL.Path == "/presigned":
			_, _ = w.Write([]byte("step 1\nstep 2\n"))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func runWatchCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL string) error {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cmd := withOutput(newRunCommand(), out)
	require.NoError(t, cmd.Flags().Set("file", cfgPath))
	require.NoError(t, cmd.Flags().Set("watch", "true"))

	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, nil, buf, buf, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srvURL))
	cmd.SetContext(ctx)
	cmd.SetOut(buf)
	return cmd.RunE(cmd, nil)
}

func TestRunWatchStreamsLogs(t *testing.T) {
	var buf bytes.Buffer
	err := runWatchCmd(t, flags.OutputText, &buf, watchServer(t, "SUCCESS").URL)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Submitted workload with Job Run ID: 777")
	assert.Contains(t, out, "View job run at: ")
	assert.Contains(t, out, "Monitoring run and streaming logs...")
	// A "Logs" divider separates the submit summary from the streamed logs.
	assert.Contains(t, out, "Logs")
	assert.Contains(t, out, "───")
	// The submitted run's logs stream through, oldest-first.
	assert.Contains(t, out, "step 1\nstep 2")
}

func TestRunWatchJSONEmitsSubmittedThenLogs(t *testing.T) {
	var buf bytes.Buffer
	err := runWatchCmd(t, flags.OutputJSON, &buf, watchServer(t, "SUCCESS").URL)
	require.NoError(t, err)

	all := buf.String()
	lines := strings.Split(strings.TrimSpace(all), "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	// First event is SUBMITTED with the run id; then STATUS + streamed LOG events.
	assert.Contains(t, lines[0], `"type":"SUBMITTED"`)
	assert.Contains(t, lines[0], `"run_id":"777"`)
	assert.Contains(t, all, `"type":"STATUS"`)
	assert.Contains(t, all, `"type":"LOG"`)
	assert.Contains(t, all, `"line":"step 1"`)
	// The last line is the closing terminal-status envelope carrying SUCCESS.
	assert.Contains(t, lines[len(lines)-1], `"status":"SUCCESS"`)
	assert.Contains(t, lines[len(lines)-1], `"run_id":"777"`)
}

func TestRunWatchJSONFailedRunTerminalEnvelope(t *testing.T) {
	var buf bytes.Buffer
	err := runWatchCmd(t, flags.OutputJSON, &buf, watchServer(t, "FAILED").URL)
	// Non-zero exit is surfaced as ErrAlreadyPrinted, but the closing envelope
	// still carries the terminal status.
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Contains(t, lines[len(lines)-1], `"status":"FAILED"`)
}

func TestRunWatchFailedRunExitsNonZero(t *testing.T) {
	var buf bytes.Buffer
	// A run that ends FAILED streams its logs but exits non-zero, surfaced as
	// ErrAlreadyPrinted (the output was already written).
	err := runWatchCmd(t, flags.OutputText, &buf, watchServer(t, "FAILED").URL)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Contains(t, buf.String(), "step 1\nstep 2")
}

func TestRunWatchDryRunSkipsSubmit(t *testing.T) {
	// --dry-run takes precedence over --watch: nothing is submitted or streamed.
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Path)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cmd := withOutput(newRunCommand(), flags.OutputText)
	require.NoError(t, cmd.Flags().Set("file", cfgPath))
	require.NoError(t, cmd.Flags().Set("watch", "true"))
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	var buf bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, nil, &buf, &buf, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL))
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, buf.String(), "Dry run")
	for _, p := range got {
		assert.NotContains(t, p, "/jobs/runs/submit", "dry-run must not submit")
		assert.NotContains(t, p, "/logs", "dry-run must not stream logs")
	}
}

func TestRunWatchJSONMLflowFallbackTerminalEnvelope(t *testing.T) {
	// Regression: through the MLflow fallback the terminal status must come from
	// the run's actual state, not the onStatusChange callback (which is
	// Bricklens-only), so a SUCCESS run isn't mislabeled FAILED in the envelope.
	var buf bytes.Buffer
	err := runWatchCmd(t, flags.OutputJSON, &buf, watchServerMLflow(t, "SUCCESS").URL)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Contains(t, lines[0], `"type":"SUBMITTED"`)
	// Logs stream through the fallback, and the closing envelope reflects the
	// real terminal status.
	assert.Contains(t, buf.String(), `"line":"step 1"`)
	assert.Contains(t, lines[len(lines)-1], `"status":"SUCCESS"`)
}

func TestRunWatchFlagRegistered(t *testing.T) {
	cmd := newRunCommand()
	f := cmd.Flags().Lookup("watch")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}
