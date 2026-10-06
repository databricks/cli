package aircmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/testserver"
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

type dryRunRequest struct {
	method string
	path   string
	body   map[string]any
}

func dryRunServer(t *testing.T, response any, requests *[]dryRunRequest) *testserver.Server {
	t.Helper()
	server := testserver.New(t)
	if requests != nil {
		server.RequestCallback = func(req *testserver.Request) {
			request := dryRunRequest{method: req.Method, path: req.URL.Path}
			if req.URL.Path == validateConfigPath {
				require.NoError(t, json.Unmarshal(req.Body, &request.body))
			}
			*requests = append(*requests, request)
		}
	}
	server.Handle("POST", validateConfigPath, func(_ testserver.Request) any { return response })
	testserver.AddDefaultHandlers(server)
	t.Cleanup(server.Close)
	return server
}

type dryRunResult struct {
	stdout string
	stderr string
	err    error
}

func runDryRunCmd(t *testing.T, timeout time.Duration, output flags.Output, srvURL, config string, flagArgs ...string) dryRunResult {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", config)
	cmd := withOutput(newRunCommandWithValidationTimeout(timeout), output)
	args := append([]string{"--file", cfgPath}, flagArgs...)
	require.NoError(t, cmd.ParseFlags(args))

	var stdout, stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), output, nil, &stdout, &stderr, "", ""))
	w := newTestWorkspaceClient(t, srvURL)
	w.Config.RetryTimeoutSeconds = 1
	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	err := cmd.RunE(cmd, nil)
	return dryRunResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func TestRunDryRunValidatesBackendWithoutMutation(t *testing.T) {
	var requests []dryRunRequest
	srv := dryRunServer(t, validateConfigResponse{}, &requests)

	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputText, srv.URL, minimalConfig, "--dry-run", "--idempotency-key", "flag-token")
	require.NoError(t, result.err)
	assert.Contains(t, result.stderr, `Dry run: configuration for "my-run" is valid; not submitting.`)

	var validations []dryRunRequest
	for _, request := range requests {
		if request.path == validateConfigPath {
			validations = append(validations, request)
		}
		assert.Contains(t, []string{"/.well-known/databricks-config", "/api/2.0/preview/scim/v2/Me", validateConfigPath}, request.path,
			"dry-run must not make requests beyond authentication, user lookup, and validation")
		assert.True(t, request.method == http.MethodGet || request.path == validateConfigPath,
			"dry-run made unexpected %s request to %s", request.method, request.path)
	}
	require.Len(t, validations, 1)
	assert.Equal(t, http.MethodPost, validations[0].method)
	assert.Len(t, validations[0].body, 2)
	assert.Contains(t, validations[0].body, "task")
	assert.Contains(t, validations[0].body, "run_options")
	task := validations[0].body["task"].(map[string]any)
	commandPath := task["deployments"].([]any)[0].(map[string]any)["command_path"].(string)
	assert.True(t, strings.HasPrefix(commandPath, "/Workspace/Users/tester@databricks.com/.air/cli_launch/my-run/my-run_"))
	assert.True(t, strings.HasSuffix(commandPath, "/command.sh"))
	runOptions := validations[0].body["run_options"].(map[string]any)
	assert.Equal(t, "flag-token", runOptions["idempotency_token"])
}

func TestRunDryRunConfigValidationHasDeadline(t *testing.T) {
	requestCanceled := make(chan struct{}, 1)
	srv := testserver.New(t)
	srv.Handle("POST", validateConfigPath, func(req testserver.Request) any {
		<-req.Context.Done()
		requestCanceled <- struct{}{}
		return nil
	})
	testserver.AddDefaultHandlers(srv)
	t.Cleanup(srv.Close)

	result := runDryRunCmd(t, 100*time.Millisecond, flags.OutputText, srv.URL, minimalConfig, "--dry-run")
	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "config validation unavailable: failed to validate config:")
	assert.Contains(t, result.err.Error(), "context deadline exceeded")
	assert.NotContains(t, result.stderr, "Dry run: configuration")
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		require.Fail(t, "dry-run validation request context was not canceled")
	}
}

func TestRunDryRunUnavailableUsesJSONErrorEnvelope(t *testing.T) {
	srv := dryRunServer(t, testserver.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       map[string]string{"error_code": "INTERNAL_ERROR", "message": "backend failed"},
	}, nil)
	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputJSON, srv.URL, minimalConfig, "--dry-run")
	require.ErrorIs(t, result.err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal([]byte(result.stdout), &got))
	assert.Equal(t, "VALIDATION_UNAVAILABLE", got.Error.Code)
	assert.Equal(t, "TRANSIENT", got.Error.Kind)
	assert.True(t, got.Error.Retryable)
	assert.Contains(t, got.Error.Message, "config validation unavailable")
	assert.NotContains(t, result.stdout, "DRY_RUN_OK")
	assert.Empty(t, result.stderr)
}

func TestRunDryRunValidatesIdempotencyTokenAfterWorkspaceValidation(t *testing.T) {
	var requests []dryRunRequest
	srv := dryRunServer(t, validateConfigResponse{}, &requests)
	tooLong := strings.Repeat("x", 65)

	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputText, srv.URL, minimalConfig, "--dry-run", "--idempotency-key", tooLong)
	require.EqualError(t, result.err, "idempotency token must be 64 characters or less, got 65")

	var validations []dryRunRequest
	for _, request := range requests {
		if request.path == validateConfigPath {
			validations = append(validations, request)
		}
	}
	require.Len(t, validations, 1)
	runOptions := validations[0].body["run_options"].(map[string]any)
	assert.Equal(t, tooLong, runOptions["idempotency_token"])
}
