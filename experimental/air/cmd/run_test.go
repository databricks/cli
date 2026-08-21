package aircmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fastMLflowPoll shrinks the MLflow-ID poll interval so tests that exercise the
// poll's retry/timeout path don't sleep a real half-second per attempt.
func fastMLflowPoll(t *testing.T) {
	t.Helper()
	orig := mlflowLinkPollInterval
	mlflowLinkPollInterval = time.Millisecond
	t.Cleanup(func() { mlflowLinkPollInterval = orig })
}

// submitServer serves a non-watch `air run` submit: runs/submit returns a run
// id, runs/get returns the run with a task, and runs/get-output returns
// getOutput (pass `{}` for "no MLflow IDs yet"). Everything else — the auth
// probe and the workspace-files upload — gets a permissive stub.
func submitServer(t *testing.T, getOutput string) *httptest.Server {
	t.Helper()
	runGet := `{"run_id": 555, "tasks": [{"run_id": 556, "attempt_number": 0}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			_, _ = w.Write([]byte(`{"run_id": 555}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runGet))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			_, _ = w.Write([]byte(getOutput))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runSubmitCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL string) error {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cmd := withOutput(newRunCommand(), out)
	require.NoError(t, cmd.Flags().Set("file", cfgPath))

	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, nil, buf, buf, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srvURL))
	cmd.SetContext(ctx)
	cmd.SetOut(buf)
	return cmd.RunE(cmd, nil)
}

func TestRunSubmitTextOutput(t *testing.T) {
	fastMLflowPoll(t)
	var buf bytes.Buffer
	// get-output carries no MLflow IDs, so the poll times out and the two MLflow
	// links are omitted — only the Job Run link is printed.
	err := runSubmitCmd(t, flags.OutputText, &buf, submitServer(t, `{}`).URL)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Submitting experiment: my-run")
	assert.Contains(t, out, "Submitted workload with Job Run ID: 555")
	assert.Contains(t, out, "View job run at: ")
	assert.Contains(t, out, "/jobs/runs/555")
	assert.Contains(t, out, "Tip: use --watch")
	assert.NotContains(t, out, "View MLflow run at:")
}

func TestRunSubmitTextOutputWithMLflowLinks(t *testing.T) {
	var buf bytes.Buffer
	srvURL := submitServer(t, `{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`).URL
	require.NoError(t, runSubmitCmd(t, flags.OutputText, &buf, srvURL))

	out := buf.String()
	assert.Contains(t, out, "View MLflow run at: ")
	assert.Contains(t, out, "/ml/experiments/exp1/runs/run1")
	assert.Contains(t, out, "View MLflow experiment at: ")
	assert.Contains(t, out, "/ml/experiments/exp1")
}

func TestRunSubmitMLflowLinksResolveOnRetry(t *testing.T) {
	fastMLflowPoll(t)
	// get-output is empty on the first poll and carries the IDs on the second,
	// exercising the poll's sleep-and-retry path.
	var getOutputCalls int
	runGet := `{"run_id": 555, "tasks": [{"run_id": 556, "attempt_number": 0}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			_, _ = w.Write([]byte(`{"run_id": 555}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runGet))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			getOutputCalls++
			if getOutputCalls == 1 {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	require.NoError(t, runSubmitCmd(t, flags.OutputText, &buf, srv.URL))

	out := buf.String()
	assert.Contains(t, out, "View MLflow run at: ")
	assert.Contains(t, out, "/ml/experiments/exp1/runs/run1")
	assert.GreaterOrEqual(t, getOutputCalls, 2, "should have polled get-output at least twice")
}

func TestRunSubmitJSONStatusPending(t *testing.T) {
	var buf bytes.Buffer
	err := runSubmitCmd(t, flags.OutputJSON, &buf, submitServer(t, `{}`).URL)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"status": "PENDING"`)
	assert.Contains(t, out, `"run_id": "555"`)
	// JSON stdout stays a clean envelope stream — no human-readable submit lines.
	assert.NotContains(t, out, "Submitting experiment")
}
