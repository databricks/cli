package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withDeadline returns the test's context with a deadline, which air deploy
// always sets from --timeout.
func withDeadline(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(fastRetries(t.Context()), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// deployServer serves registerModelPackage with responses, the staging status
// and finalizeModelPackage with their defaults, and the serving endpoints API
// from endpoints.
func deployServer(t *testing.T, endpoints *fakeEndpoints, responses ...string) (*httptest.Server, *fakeModelServing) {
	t.Helper()
	fake := newFakeModelServing(t, responses...)
	fake.endpoints = endpoints.ServeHTTP
	return fake.start(), fake
}

// runDeployCmd runs `air deploy` on config with the given flags. An empty srvURL
// runs it without a workspace client, as --dry-run does.
func runDeployCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL, config string, flagValues ...string) error {
	t.Helper()
	cmd := withOutput(newDeployCommandWithPollInterval(time.Millisecond), out)
	require.NoError(t, cmd.Flags().Set("file", writeConfigFile(t, "deploy.yaml", config)))
	for i := 0; i < len(flagValues); i += 2 {
		require.NoError(t, cmd.Flags().Set(flagValues[i], flagValues[i+1]))
	}

	// JSON tests parse buf, so stderr's progress lines go elsewhere.
	var stderr io.Writer = buf
	if out == flags.OutputJSON {
		stderr = io.Discard
	}
	ctx := cmdio.InContext(fastRetries(t.Context()), cmdio.NewIO(t.Context(), out, nil, buf, stderr, "", ""))
	if srvURL != "" {
		ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srvURL))
	}
	cmd.SetContext(ctx)
	cmd.SetOut(buf)
	return cmd.RunE(cmd, nil)
}

func TestDeployCheckpointJSON(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, fake := deployServer(t, endpoints, stagingInProgress)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.NoError(t, err)

	var got struct {
		Data deployResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, deployResult{
		Status:          "READY",
		RegisteredModel: "main.models.qwen",
		ModelVersion:    "3",
		ModelID:         "m-1",
		IdempotencyKey:  "key-1",
		Endpoint:        "qwen-chat",
		EndpointURL:     srv.URL + "/serving-endpoints/qwen-chat/invocations",
	}, got.Data)

	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.Equal(t, "key-1", bodies[0].IdempotencyKey)
	assert.Equal(t, []finalizeModelPackageRequest{{RegisteredModel: "main.models.qwen", ModelVersion: "3"}}, fake.finalizeRequests)
	assert.Equal(t, []string{"create"}, endpoints.writes)
	served := endpoints.endpoint.Config.ServedEntities[0]
	assert.Equal(t, "3", served.EntityVersion)
	assert.Equal(t, map[string]string{"LOG_LEVEL": "info", "HF_TOKEN": "{{secrets/llm/hf_token}}"}, served.EnvironmentVars)
	// scale_to_zero: false is sent, not left to the server's default.
	require.Len(t, endpoints.rawWrites, 1)
	assert.Contains(t, endpoints.rawWrites[0], `"scale_to_zero_enabled":false`)
}

func TestDeployCheckpointText(t *testing.T) {
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), stagingReady)

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig))
	out := buf.String()
	assert.Contains(t, out, "Registering a version of main.models.qwen with idempotency key ")
	assert.Contains(t, out, "Version 3 of main.models.qwen is ready.")
	assert.Contains(t, out, "Serving version 3 of main.models.qwen on endpoint qwen-chat")
	assert.Contains(t, out, "Endpoint qwen-chat serves version 3 of main.models.qwen.\n")
	assert.Contains(t, out, "Query it at: "+srv.URL+"/serving-endpoints/qwen-chat/invocations\n")
	assert.NotContains(t, out, "Warning")
}

func TestDeployWarnsWhenTheEnvironmentFailedToStage(t *testing.T) {
	environmentFailed := strings.Replace(stagingReady, `"environment_artifact": {"state": "SUCCEEDED"}`, `"environment_artifact": {"state": "FAILED"}`, 1)
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), environmentFailed)

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig))
	assert.Contains(t, buf.String(), "Warning: the version's serving environment couldn't be prepared ahead of time")
}

func TestDeployUsesTheConfigTokenThenANewKey(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), stagingReady)

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig+"idempotency_token: from-config\n"))
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig+"idempotency_token: from-config\n", "idempotency-key", "from-flag"))
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig))

	bodies := fake.requests()
	require.Len(t, bodies, 3)
	assert.Equal(t, "from-config", bodies[0].IdempotencyKey)
	assert.Equal(t, "from-flag", bodies[1].IdempotencyKey)
	assert.Regexp(t, deployIdempotencyKeyRe(), bodies[2].IdempotencyKey)
	assert.NotContains(t, []string{"from-config", "from-flag"}, bodies[2].IdempotencyKey)
}

func TestDeployPrintsTheKeyBeforeRegisteringInJSONMode(t *testing.T) {
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), conflictResponse)

	var stdout, stderr bytes.Buffer
	cmd := withOutput(newDeployCommandWithPollInterval(time.Millisecond), flags.OutputJSON)
	require.NoError(t, cmd.Flags().Set("file", writeConfigFile(t, "deploy.yaml", checkpointDeployConfig)))
	require.NoError(t, cmd.Flags().Set("idempotency-key", "key-1"))
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &stdout, &stderr, "", ""))
	cmd.SetContext(cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL)))
	require.ErrorIs(t, cmd.RunE(cmd, nil), root.ErrAlreadyPrinted)
	assert.Equal(t, "Registering a version of main.models.qwen with idempotency key key-1\n", stderr.String())
	assert.Contains(t, stdout.String(), `"code": "RESOURCE_CONFLICT"`)
}

func TestDeploySendsRetryFailedStaging(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), stagingInProgress)

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig, "retry-failed-staging", "true"))
	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.True(t, bodies[0].RetryFailedStaging)
}

func TestDeployFailedStagingSaysHowToRetry(t *testing.T) {
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), stagingFailed)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copying the weights in /Volumes/main/models/weights/qwen3_8b into version 3 of main.models.qwen failed")
	assert.Contains(t, err.Error(), " --idempotency-key key-1 --retry-failed-staging")
}

func TestDeployReportsRegistrationErrors(t *testing.T) {
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), conflictResponse)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	assert.ErrorContains(t, err, "registering a version of main.models.qwen with idempotency key key-1 failed: The idempotency key can't be used for this registration")
}

func TestDeployReportsErrorsAsJSON(t *testing.T) {
	srv, _ := deployServer(t, newFakeEndpoints(t, nil), conflictResponse)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "RESOURCE_CONFLICT", got.Error.Code)
	assert.Equal(t, "PERMANENT", got.Error.Kind)
	assert.False(t, got.Error.Retryable)

	buf.Reset()
	err = runDeployCmd(t, flags.OutputJSON, &buf, "", "registered_model: x\n", "dry-run", "true")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "INVALID_ARGS", got.Error.Code)
	assert.Contains(t, got.Error.Message, "registered_model must be a Unity Catalog model name")
}

func TestDeployChecksTheEndpointBeforeRegistering(t *testing.T) {
	other := testEntity("5")
	other.EntityName = "main.models.llama"
	srv, fake := deployServer(t, newFakeEndpoints(t, endpointServing(other, readyState)), stagingReady)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig)
	assert.ErrorContains(t, err, "endpoint qwen-chat serves version 5 of main.models.llama")
	assert.Empty(t, fake.requests())
}

func TestDeployReportsAnEndpointThatIsNotReady(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	endpoints.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}
	srv, _ := deployServer(t, endpoints, stagingReady)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	// Running air deploy again doesn't make the endpoint ready, so nothing says to.
	assert.Equal(t, "ENDPOINT_NOT_READY", got.Error.Code)
	assert.False(t, got.Error.Retryable)
	assert.NotContains(t, got.Error.Message, "run:")
}

func TestDeployTimesOutWhileCopyingTheWeights(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), stagingInProgress)
	fake.statuses = []string{statusInProgress}

	// Polls are spaced as the SDK's rate limiter allows, so the deadline, not the
	// limiter, stops the wait.
	var buf bytes.Buffer
	cmd := withOutput(newDeployCommandWithPollInterval(100*time.Millisecond), flags.OutputJSON)
	require.NoError(t, cmd.Flags().Set("file", writeConfigFile(t, "deploy.yaml", checkpointDeployConfig)))
	require.NoError(t, cmd.Flags().Set("idempotency-key", "key-1"))
	require.NoError(t, cmd.Flags().Set("timeout", "500ms"))
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &buf, io.Discard, "", ""))
	cmd.SetContext(cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL)))
	require.ErrorIs(t, cmd.RunE(cmd, nil), root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "TIMEOUT", got.Error.Code)
	assert.True(t, got.Error.Retryable)
	assert.Contains(t, got.Error.Message, "air deploy timed out while registering a version of main.models.qwen. Copying the weights continues")
	assert.Contains(t, got.Error.Message, " --idempotency-key key-1")
	assert.Len(t, fake.requests(), 1)
	assert.Empty(t, fake.finalizeRequests)
}

func TestDeployFinalizeFailureSaysHowToRetry(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), stagingSucceeded)
	fake.finalizations = []string{notFinalizableResponse}

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "finalizing version 3 of main.models.qwen failed: Version 3 of model main.models.qwen can't be finalized yet")
	assert.Contains(t, err.Error(), "To try again, run:\n  databricks air deploy -f ")
	assert.Contains(t, err.Error(), " --idempotency-key key-1")
}

func TestDeployWarnsWhenTheFinalizedEnvironmentFailedToStage(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), stagingInProgress)
	fake.finalizations = []string{strings.Replace(finalizedResponse, `"environment_artifact": {"state": "IN_PROGRESS"}`, `"environment_artifact": {"state": "FAILED"}`, 1)}

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig))
	assert.Contains(t, buf.String(), "Warning: the version's serving environment couldn't be prepared ahead of time")
}

func TestDeployRetryableRegistrationFailureSaysHowToRetry(t *testing.T) {
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), serverErrorResponse)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed: Something went wrong.\n\nTo try again, run:\n  databricks air deploy -f ")
	assert.Contains(t, err.Error(), " --idempotency-key key-1")
	assert.Len(t, fake.requests(), retry.MaxRetries+1)
}

func TestDeployEndpointFailureSaysHowToRetry(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	endpoints.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed}
	srv, _ := deployServer(t, endpoints, stagingReady)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputText, &buf, srv.URL, checkpointDeployConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serving version 3 of main.models.qwen on endpoint qwen-chat failed: ")
	assert.Contains(t, err.Error(), "To try again without registering another version, run:\n  databricks air deploy -f ")
	assert.Contains(t, err.Error(), " --idempotency-key key-1")
}

func TestDeployRegisteredVersionSkipsRegistration(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, fake := deployServer(t, endpoints)

	var buf bytes.Buffer
	err := runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, `registered_model: main.models.qwen
model: {type: registered, version: 7}
endpoint: qwen-chat
compute: {workload_type: GPU_MEDIUM, workload_size: Small}
`)
	require.NoError(t, err)
	assert.Empty(t, fake.requests())
	assert.Equal(t, "7", endpoints.endpoint.Config.ServedEntities[0].EntityVersion)
	assert.NotContains(t, buf.String(), "idempotency_key")
}

func TestDeployCodeSendsTheFiles(t *testing.T) {
	cfg, _ := writeCodeTree(t, map[string]string{"agent.py": "import tools\n", "tools/__init__.py": "x = 1\n"})
	codeReady := `{"model_id": "m-2", "registered_model": "main.agents.support", "model_version": "1", "ready": true}`
	srv, fake := deployServer(t, newFakeEndpoints(t, nil), codeReady)

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, srv.URL, codeConfigAt(cfg.CodeSource.RootPath)))
	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.Equal(t, &modelPackageCode{
		EntryFile:   "agent.py",
		EntrySource: "import tools\n",
		Files:       []modelPackageFile{{Path: "tools/__init__.py", Content: "x = 1\n"}},
		Task:        "agent/v1/responses",
		Environment: modelPackageEnvironment{Runtime: "client.5", Dependencies: []string{"mlflow==3.5.0"}},
	}, bodies[0].Code)
	assert.Nil(t, bodies[0].Checkpoint)
}

func TestDeployDryRunNeedsNoWorkspace(t *testing.T) {
	cmd := newDeployCommand()
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	assert.NoError(t, cmd.PreRunE(cmd, nil))

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputText, &buf, "", checkpointDeployConfig, "dry-run", "true"))
	assert.Contains(t, buf.String(), `Dry run: configuration for endpoint "qwen-chat" is valid; not deploying.`)

	buf.Reset()
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, "", checkpointDeployConfig, "dry-run", "true"))
	var got struct {
		Data deployResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, deployResult{Status: "DRY_RUN_OK", DryRun: true}, got.Data)
}

func TestDeployDryRunListsCodeFiles(t *testing.T) {
	cfg, _ := writeCodeTree(t, map[string]string{"agent.py": "x\n", "tools/a.py": "a\n"})

	var buf bytes.Buffer
	require.NoError(t, runDeployCmd(t, flags.OutputText, &buf, "", codeConfigAt(cfg.CodeSource.RootPath), "dry-run", "true"))
	assert.Contains(t, buf.String(), "Dry run: the model from code would include 2 file(s):\n  agent.py\n  tools/a.py\n")

	buf.Reset()
	require.NoError(t, runDeployCmd(t, flags.OutputJSON, &buf, "", codeConfigAt(cfg.CodeSource.RootPath), "dry-run", "true"))
	var got struct {
		Data deployResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, []string{"agent.py", "tools/a.py"}, got.Data.CodeFiles)
}

// codeConfigAt is codeDeployConfig with its code at root.
func codeConfigAt(root string) string {
	return strings.Replace(codeDeployConfig, "root_path: .", "root_path: "+strconv.Quote(filepath.ToSlash(root)), 1)
}

func TestDeployFlagValidation(t *testing.T) {
	registered := "registered_model: a.b.c\nmodel: {type: registered, version: 1}\nendpoint: e\ncompute: {workload_type: CPU, workload_size: Small}\n"
	tests := []struct {
		name       string
		config     string
		flagValues []string
		wantErr    string
	}{
		{"key for a registered version", registered, []string{"idempotency-key", "k"}, "--idempotency-key applies only when air deploy registers a version"},
		{"bad key", checkpointDeployConfig, []string{"idempotency-key", "a/b"}, "--idempotency-key must be 1 to 128 letters"},
		{"retry for a registered version", registered, []string{"retry-failed-staging", "true"}, "--retry-failed-staging applies only to a checkpoint"},
		{"zero timeout", checkpointDeployConfig, []string{"timeout", "0s"}, "--timeout must be positive"},
		{"empty key", checkpointDeployConfig, []string{"idempotency-key", ""}, `--idempotency-key must be 1 to 128 letters, digits, '.', '_' or '-', got ""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runDeployCmd(t, flags.OutputText, &buf, "", tt.config, append(tt.flagValues, "dry-run", "true")...)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDeployResumeCommand(t *testing.T) {
	abs, err := filepath.Abs("my deploy.yaml")
	require.NoError(t, err)
	opts := deployOptions{file: "my deploy.yaml", overrides: []string{"compute.workload_size=Medium", "task=llm/v1/chat"}}
	assert.Equal(t,
		"databricks air deploy -f "+shellquote.BashArg(filepath.ToSlash(abs))+" --override 'compute.workload_size=Medium' --override 'task=llm/v1/chat' --idempotency-key k-1 -p dev",
		deployResumeCommand("dev", opts, "k-1"))

	abs, err = filepath.Abs("d.yaml")
	require.NoError(t, err)
	assert.Equal(t, "databricks air deploy -f "+shellquote.BashArg(filepath.ToSlash(abs)), deployResumeCommand("", deployOptions{file: "d.yaml"}, ""))
}

func TestDeployStopped(t *testing.T) {
	assert.NoError(t, deployStopped(t.Context(), errors.New("boom")))

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	// The SDK's endpoint poller reports Ctrl-C as its own timeout.
	assert.ErrorIs(t, deployStopped(canceled, &retries.ErrTimedOut{}), errDeployInterrupted)
	assert.ErrorIs(t, deployStopped(canceled, context.Canceled), errDeployInterrupted)

	expired, cancelExpired := context.WithTimeout(t.Context(), -time.Second)
	defer cancelExpired()
	assert.ErrorIs(t, deployStopped(expired, context.DeadlineExceeded), errDeployTimedOut)
	assert.ErrorIs(t, deployStopped(t.Context(), &retries.ErrTimedOut{}), errDeployTimedOut)
}

func TestDeployErrorCode(t *testing.T) {
	for _, tt := range []struct {
		name      string
		err       error
		code      string
		retryable bool
	}{
		{"interrupted", fmt.Errorf("x: %w", errDeployInterrupted), "INTERRUPTED", true},
		{"timed out", fmt.Errorf("x: %w", errDeployTimedOut), "TIMEOUT", true},
		{"invalid input", withCode(codeInvalidArgs, false, errors.New("model: section is required")), "INVALID_ARGS", false},
		{"staging failed", withCode(codeStagingFailed, true, errors.New("copying failed")), "STAGING_FAILED", true},
		{"server unavailable", fmt.Errorf("x: %w", &apierr.APIError{ErrorCode: "TEMPORARILY_UNAVAILABLE", StatusCode: 503}), "TEMPORARILY_UNAVAILABLE", true},
		{"bad request", fmt.Errorf("x: %w", &apierr.APIError{ErrorCode: "INVALID_PARAMETER_VALUE", StatusCode: 400}), "INVALID_PARAMETER_VALUE", false},
		{"5xx without a body", &apierr.APIError{StatusCode: 502}, "INTERNAL_ERROR", true},
		{"error body without a code", &apierr.APIError{ErrorCode: "<nil>", StatusCode: 400}, "INTERNAL_ERROR", false},
		{"lost connection", errors.New("connection reset by peer"), "INTERNAL_ERROR", true},
		{"endpoint not ready", withCode(codeEndpointNotReady, false, errEndpointNotReady), "ENDPOINT_NOT_READY", false},
	} {
		code, retryable := deployErrorCode(tt.err)
		assert.Equal(t, tt.code, code, tt.name)
		assert.Equal(t, tt.retryable, retryable, tt.name)
	}
}

func TestDeployHelpDocumentsAField(t *testing.T) {
	var out strings.Builder
	cmd := newDeployCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config.model.type"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "config.model.type\n  One of checkpoint")
	assert.Contains(t, out.String(), "Required: yes")

	out.Reset()
	cmd = newDeployCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config"})
	require.NoError(t, cmd.Execute())
	// The coding agent skill writes run YAML, so the deploy help leaves it out.
	assert.NotContains(t, out.String(), "databricks-ai-runtime")
}
