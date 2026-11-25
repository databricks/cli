package aircmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerServer serves registerModelPackage with responses, and the staging status
// and finalizeModelPackage with their defaults.
func registerServer(t *testing.T, responses ...string) (*httptest.Server, *fakeModelServing) {
	t.Helper()
	fake := newFakeModelServing(t, responses...)
	return fake.start(), fake
}

// runModelCmd runs cmd with the given flags. An empty srvURL runs it without a
// workspace client, as --dry-run does.
func runModelCmd(t *testing.T, cmd *cobra.Command, out flags.Output, buf *bytes.Buffer, srvURL string, flagValues ...string) error {
	t.Helper()
	cmd = withOutput(cmd, out)
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

// runRegisterCmd runs `air register` on config with the given flags.
func runRegisterCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL, config string, flagValues ...string) error {
	t.Helper()
	flagValues = append([]string{"file", writeConfigFile(t, "register.yaml", config)}, flagValues...)
	return runModelCmd(t, newRegisterCommandWithPollInterval(time.Millisecond), out, buf, srvURL, flagValues...)
}

func TestRegisterCommandJSON(t *testing.T) {
	srv, fake := registerServer(t, stagingInProgress)

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "key-1"))
	var got struct {
		Data registerResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, registerResult{
		Status:         "READY",
		UCModel:        "main.models.qwen",
		ModelVersion:   "3",
		ModelID:        "m-1",
		IdempotencyKey: "key-1",
	}, got.Data)

	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.Equal(t, "key-1", bodies[0].IdempotencyKey)
	assert.Equal(t, "/Volumes/main/models/weights/qwen3_8b", bodies[0].ModelSource)
	assert.Equal(t, []finalizeModelPackageRequest{{UCModel: "main.models.qwen", ModelVersion: "3"}}, fake.finalizeRequests)
}

func TestRegisterCommandText(t *testing.T) {
	srv, _ := registerServer(t, stagingReady)

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig))
	out := buf.String()
	assert.Contains(t, out, "Registering a version of main.models.qwen with idempotency key ")
	assert.Contains(t, out, "Version 3 of main.models.qwen is ready. To serve it, run:\n  databricks air serve --uc-model main.models.qwen --endpoint-name <endpoint> --model-version 3\n")
	assert.NotContains(t, out, "Warning")
}

func TestRegisterDedicatedFromFlags(t *testing.T) {
	srv, fake := registerServer(t, stagingReady)

	var buf bytes.Buffer
	require.NoError(t, runModelCmd(t, newRegisterCommandWithPollInterval(time.Millisecond), flags.OutputJSON, &buf, srv.URL,
		"model-source", "/Volumes/main/models/weights/gemma", "uc-model", "main.models.gemma"))
	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.Nil(t, bodies[0].Engine)
	assert.Equal(t, "/Volumes/main/models/weights/gemma", bodies[0].ModelSource)
	assert.Equal(t, "main.models.gemma", bodies[0].UCModel)
}

func TestRegisterWarnsWhenTheEnvironmentFailedToStage(t *testing.T) {
	environmentFailed := strings.Replace(stagingReady, `"environment_artifact": {"state": "SUCCEEDED"}`, `"environment_artifact": {"state": "FAILED"}`, 1)
	srv, _ := registerServer(t, environmentFailed)

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig))
	assert.Contains(t, buf.String(), "Warning: the version's serving environment couldn't be prepared ahead of time")

	buf.Reset()
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig))
	assert.Contains(t, buf.String(), `"environment_staging_failed": true`)
}

func TestRegisterWarnsWhenTheFinalizedEnvironmentFailedToStage(t *testing.T) {
	srv, fake := registerServer(t, stagingInProgress)
	fake.finalizations = []string{strings.Replace(finalizedResponse, `"environment_artifact": {"state": "IN_PROGRESS"}`, `"environment_artifact": {"state": "FAILED"}`, 1)}

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig))
	assert.Contains(t, buf.String(), "Warning: the version's serving environment couldn't be prepared ahead of time")
}

func TestRegisterUsesTheFlagKeyElseANewOne(t *testing.T) {
	srv, fake := registerServer(t, stagingReady)

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "from-flag"))
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig))
	bodies := fake.requests()
	require.Len(t, bodies, 2)
	assert.Equal(t, "from-flag", bodies[0].IdempotencyKey)
	assert.Regexp(t, registrationKeyRe(), bodies[1].IdempotencyKey)
	assert.NotEqual(t, "from-flag", bodies[1].IdempotencyKey)
}

func TestRegisterPrintsTheKeyBeforeRegisteringInJSONMode(t *testing.T) {
	srv, _ := registerServer(t, conflictResponse)

	var stdout, stderr bytes.Buffer
	cmd := withOutput(newRegisterCommandWithPollInterval(time.Millisecond), flags.OutputJSON)
	require.NoError(t, cmd.Flags().Set("file", writeConfigFile(t, "register.yaml", commandRegisterConfig)))
	require.NoError(t, cmd.Flags().Set("idempotency-key", "key-1"))
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &stdout, &stderr, "", ""))
	cmd.SetContext(cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL)))
	require.ErrorIs(t, cmd.RunE(cmd, nil), root.ErrAlreadyPrinted)
	assert.Equal(t, "Registering a version of main.models.qwen with idempotency key key-1\n", stderr.String())
	assert.Contains(t, stdout.String(), `"code": "RESOURCE_CONFLICT"`)
}

func TestRegisterSendsRetryFailedStaging(t *testing.T) {
	srv, fake := registerServer(t, stagingInProgress)

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig, "retry-failed-staging", "true"))
	bodies := fake.requests()
	require.Len(t, bodies, 1)
	assert.True(t, bodies[0].RetryFailedStaging)
}

func TestRegisterFailedStagingSaysHowToRetry(t *testing.T) {
	srv, _ := registerServer(t, stagingFailed)

	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copying the weights in /Volumes/main/models/weights/qwen3_8b into version 3 of main.models.qwen failed")
	assert.Contains(t, err.Error(), " --idempotency-key key-1 --retry-failed-staging")
}

func TestRegisterReportsRegistrationErrors(t *testing.T) {
	srv, _ := registerServer(t, conflictResponse)

	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "key-1")
	assert.ErrorContains(t, err, "registering a version of main.models.qwen with idempotency key key-1 failed: The idempotency key can't be used for this registration")
}

func TestRegisterReportsErrorsAsJSON(t *testing.T) {
	srv, _ := registerServer(t, conflictResponse)

	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputJSON, &buf, srv.URL, commandRegisterConfig)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "RESOURCE_CONFLICT", got.Error.Code)
	assert.Equal(t, "PERMANENT", got.Error.Kind)
	assert.False(t, got.Error.Retryable)

	buf.Reset()
	err = runRegisterCmd(t, flags.OutputJSON, &buf, "", "model_source: /Volumes/a/b/c/w\nuc_model: x\n", "dry-run", "true")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "INVALID_ARGS", got.Error.Code)
	assert.Contains(t, got.Error.Message, "uc_model must be a Unity Catalog model name")
}

func TestRegisterTimesOutWhileCopyingTheWeights(t *testing.T) {
	srv, fake := registerServer(t, stagingInProgress)
	fake.statuses = []string{statusInProgress}

	// Polls are spaced as the SDK's rate limiter allows, so the deadline, not the
	// limiter, stops the wait.
	var buf bytes.Buffer
	cmd := withOutput(newRegisterCommandWithPollInterval(100*time.Millisecond), flags.OutputJSON)
	require.NoError(t, cmd.Flags().Set("file", writeConfigFile(t, "register.yaml", commandRegisterConfig)))
	require.NoError(t, cmd.Flags().Set("idempotency-key", "key-1"))
	require.NoError(t, cmd.Flags().Set("timeout", "500ms"))
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &buf, io.Discard, "", ""))
	cmd.SetContext(cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL)))
	require.ErrorIs(t, cmd.RunE(cmd, nil), root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "TIMEOUT", got.Error.Code)
	assert.True(t, got.Error.Retryable)
	assert.Contains(t, got.Error.Message, "air register timed out while registering a version of main.models.qwen. Copying the weights continues")
	assert.Contains(t, got.Error.Message, " --idempotency-key key-1")
	assert.Len(t, fake.requests(), 1)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterFinalizeFailureSaysHowToRetry(t *testing.T) {
	srv, fake := registerServer(t, stagingSucceeded)
	fake.finalizations = []string{notFinalizableResponse}

	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "finalizing version 3 of main.models.qwen failed: Version 3 of model main.models.qwen can't be finalized yet")
	assert.Contains(t, err.Error(), "To try again, run:\n  databricks air register -f ")
	assert.Contains(t, err.Error(), " --idempotency-key key-1")
}

func TestRegisterRetryableRegistrationFailureSaysHowToRetry(t *testing.T) {
	srv, fake := registerServer(t, serverErrorResponse)

	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputText, &buf, srv.URL, commandRegisterConfig, "idempotency-key", "key-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed: Something went wrong.\n\nTo try again, run:\n  databricks air register -f ")
	assert.Contains(t, err.Error(), " --idempotency-key key-1")
	assert.Len(t, fake.requests(), retry.MaxRetries+1)
}

func TestRegisterDryRunNeedsNoWorkspace(t *testing.T) {
	cmd := newRegisterCommand()
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	assert.NoError(t, cmd.PreRunE(cmd, nil))

	var buf bytes.Buffer
	require.NoError(t, runRegisterCmd(t, flags.OutputText, &buf, "", commandRegisterConfig, "dry-run", "true"))
	out := buf.String()
	assert.Contains(t, out, "Dry run: would register a new version of main.models.qwen from /Volumes/main/models/weights/qwen3_8b, served by its command:")
	assert.Contains(t, out, `"command": "python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080"`)
	assert.Contains(t, out, "Dry run: configuration is valid; nothing was changed.")

	buf.Reset()
	require.NoError(t, runRegisterCmd(t, flags.OutputJSON, &buf, "", dedicatedRegisterConfig, "dry-run", "true"))
	var got struct {
		Data registerResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "DRY_RUN_OK", got.Data.Status)
	assert.Equal(t, &registerPlan{Variant: "dedicated", Request: registerModelPackageRequest{ModelSource: "/Volumes/main/models/weights/qwen3_8b", UCModel: "main.models.qwen"}}, got.Data.Plan)
}

func TestRegisterFlagValidation(t *testing.T) {
	tests := []struct {
		name       string
		flagValues []string
		wantErr    string
	}{
		{"zero timeout", []string{"timeout", "0s"}, "--timeout must be positive"},
		{"bad key", []string{"idempotency-key", "a/b"}, "--idempotency-key must be 1 to 128 letters"},
		{"empty key", []string{"idempotency-key", ""}, `--idempotency-key must be 1 to 128 letters, digits, '.', '_' or '-', got ""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runRegisterCmd(t, flags.OutputText, &buf, "", commandRegisterConfig, append(tt.flagValues, "dry-run", "true")...)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRegisterRefusesAnEmptyModelSourceFlag(t *testing.T) {
	var buf bytes.Buffer
	err := runRegisterCmd(t, flags.OutputText, &buf, "", commandRegisterConfig, "model-source", "", "dry-run", "true")
	assert.ErrorContains(t, err, "--model-source cannot be empty")
}

func TestRegisterResumeCommand(t *testing.T) {
	abs, err := filepath.Abs("my register.yaml")
	require.NoError(t, err)
	opts := registerOptions{file: "my register.yaml", overrides: []string{"engine.task=llm/v1/chat"}, ucModel: "main.models.other"}
	assert.Equal(t,
		"databricks air register -f "+shellquote.BashArg(filepath.ToSlash(abs))+" --override 'engine.task=llm/v1/chat' --uc-model main.models.other --idempotency-key k-1 -p dev",
		registerResumeCommand("dev", opts, "k-1"))

	assert.Equal(t,
		"databricks air register --model-source /Volumes/a/b/c/w --uc-model a.b.c --idempotency-key k-2",
		registerResumeCommand("", registerOptions{modelSource: "/Volumes/a/b/c/w", ucModel: "a.b.c"}, "k-2"))
}

func TestRegisterHelpDocumentsAField(t *testing.T) {
	var out strings.Builder
	cmd := newRegisterCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config.model_source"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "config.model_source\n  Unity Catalog volume directory")
	assert.Contains(t, out.String(), "Required: yes")

	out.Reset()
	cmd = newRegisterCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config"})
	require.NoError(t, cmd.Execute())
	// The coding agent skill writes run YAML, so the register help leaves it out.
	assert.NotContains(t, out.String(), "databricks-ai-runtime")
}

func TestRegisterHelpNamesTheVersionsType(t *testing.T) {
	var out strings.Builder
	cmd := newRegisterCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config.engine.environment.version"})
	require.NoError(t, cmd.Execute())
	// version is a *stringOrInt, which documents as its union, not as an object.
	assert.Contains(t, out.String(), "string or int")
	assert.NotContains(t, out.String(), "object")
}
