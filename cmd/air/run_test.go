package aircmd

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	assert.Contains(t, out, `"existing": false`)
	// JSON stdout stays a clean envelope stream — no human-readable submit lines.
	assert.NotContains(t, out, "Submitting experiment")
}

// idempotentSubmitServer runs the fake workspace, whose runs/submit returns the
// run a repeated idempotency token started.
func idempotentSubmitServer(t *testing.T) string {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("POST", validateConfigPath, func(testserver.Request) any {
		return validateConfigResponse{}
	})
	testserver.AddDefaultHandlers(server)
	return server.URL
}

func runIdempotentSubmit(t *testing.T, srvURL string, out flags.Output, config string, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	cfgPath := writeConfigFile(t, "run.yaml", config)
	cmd := withOutput(newRunCommand(), out)
	require.NoError(t, cmd.ParseFlags(append([]string{"--file", cfgPath}, args...)))

	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, nil, &buf, &buf, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srvURL))
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)
	require.NoError(t, cmd.RunE(cmd, nil))
	return buf.String()
}

func TestRunSubmitIdempotencyKeyReturnsExistingRun(t *testing.T) {
	srvURL := idempotentSubmitServer(t)

	first := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--idempotency-key", "flag-key")
	assert.Contains(t, first, "Submitted workload with Job Run ID: ")
	assert.NotContains(t, first, "existing")

	second := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--idempotency-key", "flag-key")
	assert.Contains(t, second, "Returned existing workload with Job Run ID: ")
	assert.Contains(t, second, `(idempotency key "flag-key" was already used)`)
	assert.NotContains(t, second, "Submitted workload")
	assert.Contains(t, second, "View job run at: ")

	other := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--idempotency-key", "other-key")
	assert.Contains(t, other, "Submitted workload with Job Run ID: ")
}

func TestRunSubmitYAMLIdempotencyTokenReturnsExistingRun(t *testing.T) {
	srvURL := idempotentSubmitServer(t)
	config := minimalConfig + "idempotency_token: yaml-key\n"

	first := runIdempotentSubmit(t, srvURL, flags.OutputText, config)
	assert.Contains(t, first, "Submitted workload with Job Run ID: ")

	second := runIdempotentSubmit(t, srvURL, flags.OutputText, config)
	assert.Contains(t, second, `(idempotency key "yaml-key" was already used)`)
	assert.NotContains(t, second, "Submitted workload")
}

func TestRunSubmitJSONReportsExistingRun(t *testing.T) {
	srvURL := idempotentSubmitServer(t)

	first := decodeRunResult(t, runIdempotentSubmit(t, srvURL, flags.OutputJSON, minimalConfig, "--idempotency-key", "json-key"))
	require.NotNil(t, first.Existing)
	assert.False(t, *first.Existing)

	second := decodeRunResult(t, runIdempotentSubmit(t, srvURL, flags.OutputJSON, minimalConfig, "--idempotency-key", "json-key"))
	require.NotNil(t, second.Existing)
	assert.True(t, *second.Existing)
	assert.Equal(t, first.RunID, second.RunID)
	assert.Equal(t, "PENDING", second.Status)
}

// earlierLaunchRunGet is a terminated run whose command path is from an earlier
// submit's launch directory, so a submit with a user key sees it as existing.
const earlierLaunchRunGet = `{
	"run_id": 777,
	"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
	"tasks": [{
		"run_id": 778,
		"attempt_number": 0,
		"ai_runtime_task": {"deployments": [{"command_path": "/Users/u@example.com/.air/cli_launch/my-run/my-run_0123456789abcdef/command.sh"}]}
	}]
}`

// runGetReply answers runs/get; submittedCommandPath is the command path the
// preceding runs/submit sent.
type runGetReply func(w http.ResponseWriter, r *http.Request, submittedCommandPath string)

func staticRunGet(status int, body string) runGetReply {
	return func(w http.ResponseWriter, _ *http.Request, _ string) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// existingRunServer serves `air run` with or without --watch. runs/get replies
// via runGet; the returned counter tracks its calls.
func existingRunServer(t *testing.T, runGet runGetReply) (string, *atomic.Int32) {
	t.Helper()
	var runGets atomic.Int32
	var submittedCommandPath atomic.Value
	submittedCommandPath.Store("")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/runs/submit"):
			var req jobs.SubmitRun
			if json.NewDecoder(r.Body).Decode(&req) == nil && len(req.Tasks) > 0 && req.Tasks[0].AiRuntimeTask != nil && len(req.Tasks[0].AiRuntimeTask.Deployments) > 0 {
				submittedCommandPath.Store(req.Tasks[0].AiRuntimeTask.Deployments[0].CommandPath)
			}
			_, _ = w.Write([]byte(`{"run_id": 777}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			runGets.Add(1)
			runGet(w, r, submittedCommandPath.Load().(string))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 1700000002000000000, "body": "step 2", "node_index": 0},
				{"time_unix_nano": 1700000001000000000, "body": "step 1", "node_index": 0}
			]}`))
		default:
			_, _ = w.Write([]byte(`{"userName": "u@example.com", "workspace_id": 1}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &runGets
}

// decodeRunResult returns the data of a non-watch `air run -o json` envelope.
func decodeRunResult(t *testing.T, out string) runResult {
	t.Helper()
	var envelope struct {
		Data runResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &envelope), out)
	return envelope.Data
}

func TestRunWatchExistingRunPrintsBeforeLogs(t *testing.T) {
	srvURL, _ := existingRunServer(t, staticRunGet(http.StatusOK, earlierLaunchRunGet))

	out := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--watch", "--idempotency-key", "watch-key")
	existingLine := `Returned existing workload with Job Run ID: 777 (idempotency key "watch-key" was already used)`
	require.Contains(t, out, existingLine)
	require.Contains(t, out, "step 1\nstep 2")
	assert.Less(t, strings.Index(out, existingLine), strings.Index(out, "step 1"))
	assert.NotContains(t, out, "Submitted workload")
}

func TestRunExistingRunLookupByOutputMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  flags.Output
		args []string
		// wantLookups is the runs/get calls a user key adds over a generated token.
		wantLookups int32
	}{
		{name: "text", out: flags.OutputText, wantLookups: 1},
		{name: "json", out: flags.OutputJSON, wantLookups: 1},
		{name: "watch text", out: flags.OutputText, args: []string{"--watch"}, wantLookups: 1},
		// The --watch JSON events don't report existing, so there is no lookup.
		{name: "watch json", out: flags.OutputJSON, args: []string{"--watch"}, wantLookups: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generatedURL, generatedRunGets := existingRunServer(t, staticRunGet(http.StatusOK, earlierLaunchRunGet))
			runIdempotentSubmit(t, generatedURL, tc.out, minimalConfig, tc.args...)

			userKeyURL, userKeyRunGets := existingRunServer(t, staticRunGet(http.StatusOK, earlierLaunchRunGet))
			runIdempotentSubmit(t, userKeyURL, tc.out, minimalConfig, append(tc.args, "--idempotency-key", "mode-key")...)

			assert.Equal(t, tc.wantLookups, userKeyRunGets.Load()-generatedRunGets.Load())
		})
	}
}

func TestRunSubmitExistingRunMatchesLaunchDirAcrossPathPrefixes(t *testing.T) {
	// The backend echoes this submit's own command path in another prefix form.
	srvURL, _ := existingRunServer(t, func(w http.ResponseWriter, _ *http.Request, submitted string) {
		echoed := strings.TrimPrefix(submitted, "/Workspace")
		require.NotEqual(t, submitted, echoed)
		_, _ = fmt.Fprintf(w, `{"run_id": 777, "tasks": [{"run_id": 778, "ai_runtime_task": {"deployments": [{"command_path": %q}]}}]}`, echoed)
	})

	out := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--idempotency-key", "prefix-key")
	assert.Contains(t, out, "Submitted workload with Job Run ID: 777")

	result := decodeRunResult(t, runIdempotentSubmit(t, srvURL, flags.OutputJSON, minimalConfig, "--idempotency-key", "prefix-key"))
	require.NotNil(t, result.Existing)
	assert.False(t, *result.Existing)
}

func TestRunSubmitExistingRunLookupReportsUnknown(t *testing.T) {
	for _, tc := range []struct {
		name   string
		runGet runGetReply
	}{
		{
			name:   "runs/get fails",
			runGet: staticRunGet(http.StatusNotFound, `{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "run not found"}`),
		},
		{
			name: "runs/get times out",
			runGet: func(_ http.ResponseWriter, r *http.Request, _ string) {
				<-r.Context().Done()
			},
		},
		{
			name:   "no deployment",
			runGet: staticRunGet(http.StatusOK, `{"run_id": 777, "tasks": [{"run_id": 778, "ai_runtime_task": {}}]}`),
		},
		{
			name:   "empty command path",
			runGet: staticRunGet(http.StatusOK, `{"run_id": 777, "tasks": [{"run_id": 778, "ai_runtime_task": {"deployments": [{}]}}]}`),
		},
		{
			name:   "unrecognized command path",
			runGet: staticRunGet(http.StatusOK, `{"run_id": 777, "tasks": [{"run_id": 778, "ai_runtime_task": {"deployments": [{"command_path": "/Users/u@example.com/train.sh"}]}}]}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultTimeout := existingRunLookupTimeout
			existingRunLookupTimeout = 50 * time.Millisecond
			t.Cleanup(func() { existingRunLookupTimeout = defaultTimeout })
			srvURL, runGets := existingRunServer(t, tc.runGet)

			out := runIdempotentSubmit(t, srvURL, flags.OutputText, minimalConfig, "--idempotency-key", "unknown-key")
			assert.Contains(t, out, "Submitted workload with Job Run ID: 777")
			assert.NotContains(t, out, "Returned existing workload")

			jsonOut := runIdempotentSubmit(t, srvURL, flags.OutputJSON, minimalConfig, "--idempotency-key", "unknown-key")
			assert.Nil(t, decodeRunResult(t, jsonOut).Existing)
			assert.NotContains(t, jsonOut, `"existing"`)
			assert.Equal(t, int32(2), runGets.Load())
		})
	}
}

func TestRunSubmitContainersSkipsExistingRunLookup(t *testing.T) {
	srvURL, runGets := existingRunServer(t, staticRunGet(http.StatusOK, earlierLaunchRunGet))
	config := `
experiment_name: my-run
compute:
  accelerator_type: GPU_1xH100
  num_accelerators: 1
containers:
  - name: inference
    command: python infer.py
    ranks: [0]
    unity_catalog_image: main.ml.inference:v1
`

	out := runIdempotentSubmit(t, srvURL, flags.OutputText, config, "--idempotency-key", "container-key")
	assert.Contains(t, out, "Submitted workload with Job Run ID: 777")

	jsonOut := runIdempotentSubmit(t, srvURL, flags.OutputJSON, config, "--idempotency-key", "container-key")
	assert.NotContains(t, jsonOut, `"existing"`)
	assert.Zero(t, runGets.Load())
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
	w.Config.RetryTimeoutSeconds = 5
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
	assert.Len(t, validations[0].body, 3)
	assert.Contains(t, validations[0].body, "task")
	assert.Contains(t, validations[0].body, "run_options")
	assert.Equal(t, map[string]any{"environment_version": "6"}, validations[0].body["environment"])
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

func TestRunDryRunRetriesUnavailableResponseOnce(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests []dryRunRequest
			srv := dryRunServer(t, testserver.Response{
				StatusCode: status,
				Body:       map[string]string{"error_code": "TEMPORARILY_UNAVAILABLE", "message": "try again"},
			}, &requests)

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

			validationRequests := 0
			for _, request := range requests {
				if request.path == validateConfigPath {
					validationRequests++
				}
			}
			assert.Equal(t, dryRunValidationMaxAttempts, validationRequests)
		})
	}
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
