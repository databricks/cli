package aircmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type submitRequestCounts struct {
	runGet       atomic.Int32
	runGetOutput atomic.Int32
}

// submitServer serves a non-watch `air run` submit. It records any post-submit
// run lookups so tests can assert that the command returns without MLflow polling.
// Everything except runs/submit gets a permissive stub.
func submitServer(t *testing.T) (*httptest.Server, *submitRequestCounts) {
	t.Helper()
	counts := &submitRequestCounts{}
	runGet := `{"run_id": 555, "tasks": [{"run_id": 556, "attempt_number": 0}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			_, _ = w.Write([]byte(`{"run_id": 555}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			counts.runGet.Add(1)
			_, _ = w.Write([]byte(runGet))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			counts.runGetOutput.Add(1)
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "exp1", "mlflow_run_id": "run1"}}`))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, counts
}

func runSubmitCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL string) error {
	return runSubmitCmdWithProfile(t, out, buf, srvURL, "")
}

func runSubmitCmdWithProfile(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL, profile string) error {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cmd := withOutput(newRunCommand(), out)
	require.NoError(t, cmd.Flags().Set("file", cfgPath))

	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, nil, buf, buf, "", ""))
	w := newTestWorkspaceClient(t, srvURL)
	w.Config.Profile = profile
	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	cmd.SetContext(ctx)
	cmd.SetOut(buf)
	return cmd.RunE(cmd, nil)
}

func TestRunSubmitTextOutput(t *testing.T) {
	var buf bytes.Buffer
	srv, counts := submitServer(t)
	err := runSubmitCmd(t, flags.OutputText, &buf, srv.URL)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Submitting experiment: my-run")
	assert.Contains(t, out, "Submitted workload with Job Run ID: 555")
	assert.Contains(t, out, "View job run at: ")
	assert.Contains(t, out, "/jobs/runs/555")
	assert.Contains(t, out, "Tip: use --watch when submitting a run to stream logs to your terminal.")
	assert.Contains(t, out, "Stream logs after submission using:")
	assert.Contains(t, out, "databricks air logs 555")
	assert.NotContains(t, out, "View MLflow run at:")
	assert.Zero(t, counts.runGet.Load(), "bare submission must not poll runs/get")
	assert.Zero(t, counts.runGetOutput.Load(), "bare submission must not poll runs/get-output")
}

func TestWarnExperimentalContainers(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, nil, &stdout, &stderr, "", ""))

	warnExperimentalContainers(ctx, &runConfig{})
	assert.Empty(t, stderr.String())

	warnExperimentalContainers(ctx, &runConfig{Containers: []containerConfig{{Name: "inference"}}})
	assert.Equal(t, experimentalContainersWarning+"\n", stderr.String())
}

func TestRunSubmitTextOutputIncludesProfileInLogsCommand(t *testing.T) {
	var buf bytes.Buffer
	srv, _ := submitServer(t)
	err := runSubmitCmdWithProfile(t, flags.OutputText, &buf, srv.URL, "team profile")
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "databricks air logs 555 -p 'team profile'")
}

func TestAirLogsCommand(t *testing.T) {
	assert.Equal(t, "databricks air logs 123", airLogsCommand("", "123"))
	assert.Equal(t, "databricks air logs 123 -p profile-name", airLogsCommand("profile-name", "123"))
	assert.Equal(t, "databricks air logs 123 -p 'team profile'", airLogsCommand("team profile", "123"))
}

func TestAirGetCommand(t *testing.T) {
	assert.Equal(t, "databricks air get 123", airGetCommand("", "123"))
	assert.Equal(t, "databricks air get 123 -p profile-name", airGetCommand("profile-name", "123"))
	assert.Equal(t, "databricks air get 123 -p 'team profile'", airGetCommand("team profile", "123"))
}

func TestRunSubmitJSONStatusPending(t *testing.T) {
	var buf bytes.Buffer
	srv, _ := submitServer(t)
	err := runSubmitCmd(t, flags.OutputJSON, &buf, srv.URL)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"status": "PENDING"`)
	assert.Contains(t, out, `"run_id": "555"`)
	// JSON stdout stays a clean envelope stream — no human-readable submit lines.
	assert.NotContains(t, out, "Submitting experiment")
}
