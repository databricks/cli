package aircmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

type dryRunRequest struct {
	method string
	path   string
	body   map[string]any
}

// dryRunServer records workspace reads and serves one validation response.
func dryRunServer(t *testing.T, status int, response string, requests *[]dryRunRequest) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := dryRunRequest{method: r.Method, path: r.URL.Path}
		if r.URL.Path == "/.well-known/databricks-config" {
			*requests = append(*requests, request)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == validateConfigPath {
			if err := json.NewDecoder(r.Body).Decode(&request.body); err != nil {
				t.Errorf("decode validation request: %v", err)
			}
			*requests = append(*requests, request)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(response))
			return
		}
		*requests = append(*requests, request)
		_, _ = w.Write([]byte(`{"userName":"u@example.com","workspace_id":1}`))
	}))
	t.Cleanup(srv.Close)
	return srv
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
	srv := dryRunServer(t, http.StatusOK, `{}`, &requests)
	config := minimalConfig + `
environment:
  dependencies:
    - numpy
code_source:
  type: snapshot
  snapshot:
    root_path: .
max_retries: 2
usage_policy_name: team-policy
permissions:
  - group_name: users
    level: CAN_VIEW
`

	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputText, srv.URL, config, "--dry-run", "--idempotency-key", "flag-token")
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
	assert.True(t, strings.HasPrefix(commandPath, "/Workspace/Users/u@example.com/.air/cli_launch/my-run/my-run_"))
	assert.True(t, strings.HasSuffix(commandPath, "/command.sh"))
	runOptions := validations[0].body["run_options"].(map[string]any)
	assert.Equal(t, "flag-token", runOptions["idempotency_token"])
}

func TestRunDryRunConfigValidationHasDeadline(t *testing.T) {
	requestCanceled := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			w.WriteHeader(http.StatusNotFound)
		case "/api/2.0/preview/scim/v2/Me":
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		case validateConfigPath:
			_, _ = io.Copy(io.Discard, r.Body)
			<-r.Context().Done()
			requestCanceled <- struct{}{}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	result := runDryRunCmd(t, 100*time.Millisecond, flags.OutputText, srv.URL, minimalConfig, "--dry-run")
	require.NoError(t, result.err)
	assert.Contains(t, result.stderr, "only local validation was performed.")
	assert.Contains(t, result.stderr, `Dry run: local validation passed for "my-run"; not submitting.`)
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		require.Fail(t, "dry-run validation request context was not canceled")
	}
}

func TestRunDryRunFailsOnAuthenticationFailure(t *testing.T) {
	var requests []dryRunRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, dryRunRequest{method: r.Method, path: r.URL.Path})
		if r.URL.Path == "/.well-known/databricks-config" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error_code":"UNAUTHENTICATED","message":"log in"}`))
	}))
	t.Cleanup(srv.Close)

	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputText, srv.URL, minimalConfig, "--dry-run")
	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "failed to resolve current user")
	assert.NotContains(t, result.stderr, "only local validation was performed")
	for _, request := range requests {
		assert.NotEqual(t, validateConfigPath, request.path)
	}
}

func TestRunDryRunIncompleteKeepsJSONOutputContract(t *testing.T) {
	var requests []dryRunRequest
	srv := dryRunServer(t, http.StatusInternalServerError, `{"error_code":"INTERNAL_ERROR","message":"backend failed"}`, &requests)
	result := runDryRunCmd(t, dryRunValidationTimeout, flags.OutputJSON, srv.URL, minimalConfig, "--dry-run")
	require.NoError(t, result.err)
	var got struct {
		Data runResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.stdout), &got))
	assert.Equal(t, "DRY_RUN_OK", got.Data.Status)
	assert.True(t, got.Data.DryRun)
	assert.Contains(t, result.stderr, "only local validation was performed")
}

func TestRunDryRunValidatesEffectiveIdempotencyTokenBeforeLocalFallback(t *testing.T) {
	var requests []dryRunRequest
	srv := dryRunServer(t, http.StatusOK, `{}`, &requests)
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
