package aircmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modelVersionsJSON lists the versions of the test model: 4 is the newest READY one.
const modelVersionsJSON = `{"model_versions": [
	{"version": 2, "status": "READY"},
	{"version": 5, "status": "PENDING_REGISTRATION"},
	{"version": 4, "status": "READY"}
]}`

// fakeServing serves the test model's versions, as Unity Catalog lists them, each
// version's MLmodel through the Files API, and the serving endpoints API from
// endpoints. It counts the version listings.
type fakeServing struct {
	endpoints *fakeEndpoints
	mlmodel   string
	// versions is the version listing, modelVersionsJSON unless a test sets one.
	versions string
	mu       sync.Mutex
	listings int
}

func (f *fakeServing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/versions"):
		f.mu.Lock()
		f.listings++
		f.mu.Unlock()
		versions := f.versions
		if versions == "" {
			versions = modelVersionsJSON
		}
		_, _ = w.Write([]byte(versions))
	case strings.HasPrefix(r.URL.Path, "/api/2.0/fs/files/Models/main/models/qwen/") && strings.HasSuffix(r.URL.Path, "/MLmodel") && f.mlmodel != "":
		_, _ = w.Write([]byte(f.mlmodel))
	case strings.HasPrefix(r.URL.Path, "/api/2.0/fs/files/"):
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code": "NOT_FOUND", "message": "The file being accessed is not found."}`))
	default:
		f.endpoints.ServeHTTP(w, r)
	}
}

// serveServer serves versions of the test model whose MLmodel is mlmodel, and the
// serving endpoints API from endpoints.
func serveServer(t *testing.T, endpoints *fakeEndpoints, mlmodel string) (*httptest.Server, *fakeServing) {
	t.Helper()
	fake := &fakeServing{endpoints: endpoints, mlmodel: mlmodel}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return srv, fake
}

// runServeCmd runs `air serve` on config with the given flags.
func runServeCmd(t *testing.T, out flags.Output, buf *bytes.Buffer, srvURL, config string, flagValues ...string) error {
	t.Helper()
	flagValues = append([]string{"file", writeConfigFile(t, "serve.yaml", config)}, flagValues...)
	return runModelCmd(t, newServeCommand(), out, buf, srvURL, flagValues...)
}

func serveJSON(t *testing.T, buf *bytes.Buffer) serveResult {
	t.Helper()
	var got struct {
		Data serveResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	return got.Data
}

func TestServeNewestReadyVersionOfACommand(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig, "watch", "true"))
	assert.Equal(t, serveResult{
		Status:       "READY",
		UCModel:      "main.models.qwen",
		ModelVersion: "4",
		Endpoint:     "qwen-chat",
		EndpointURL:  srv.URL + "/serving-endpoints/qwen-chat/invocations",
	}, serveJSON(t, &buf))
	assert.Equal(t, []string{"create"}, endpoints.writes)
	served := endpoints.endpoint.Config.ServedEntities[0]
	assert.Equal(t, "4", served.EntityVersion)
	assert.Equal(t, "GPU_MEDIUM", served.WorkloadType)
	assert.Equal(t, 4, served.MinProvisionedConcurrency)
	assert.Equal(t, 4, served.MaxProvisionedConcurrency)
	// scale_to_zero_enabled: false is sent, not left to the server's default, and the
	// scale goes as a concurrency range instead of a workload size.
	require.Len(t, endpoints.rawWrites, 1)
	assert.Contains(t, endpoints.rawWrites[0], `"scale_to_zero_enabled":false`)
	assert.NotContains(t, endpoints.rawWrites[0], "workload_size")
	assert.NotContains(t, endpoints.rawWrites[0], "environment_vars")
}

func TestServeText(t *testing.T) {
	srv, _ := serveServer(t, newFakeEndpoints(t, nil), commandMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputText, &buf, srv.URL, minimalServeConfig, "watch", "true"))
	out := buf.String()
	assert.Contains(t, out, "Serving version 4, the newest READY version of main.models.qwen.")
	assert.Contains(t, out, "Serving version 4 of main.models.qwen on endpoint qwen-chat")
	assert.Contains(t, out, "Endpoint qwen-chat serves version 4 of main.models.qwen.\nQuery it at: "+srv.URL+"/serving-endpoints/qwen-chat/invocations\n")
}

func TestServeWithoutWatchReturnsWhileTheEndpointUpdates(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	endpoints.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress}
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig))
	got := serveJSON(t, &buf)
	assert.Equal(t, "UPDATING", got.Status)
	assert.Equal(t, "4", got.ModelVersion)
	assert.Equal(t, []string{"create"}, endpoints.writes)

	srv, _ = serveServer(t, newFakeEndpoints(t, nil), commandMLmodel)
	buf.Reset()
	require.NoError(t, runServeCmd(t, flags.OutputText, &buf, srv.URL, minimalServeConfig))
	assert.Contains(t, buf.String(), "Endpoint qwen-chat is updating to serve version 4 of main.models.qwen, at "+srv.URL+"/serving-endpoints/qwen-chat/invocations once it's ready. To wait for it, run:\n  databricks air serve -f ")
	assert.Contains(t, buf.String(), " --model-version 4 --watch\n")
}

func TestServeADedicatedCheckpoint(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, dedicatedMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig, "watch", "true"))
	assert.Equal(t, []string{"create dedicated"}, endpoints.writes)
	require.Len(t, endpoints.rawWrites, 1)
	assert.JSONEq(t, `{"name": "qwen-chat", "config": {"served_entities": [{
		"entity_name": "main.models.qwen",
		"entity_version": "4",
		"provisioned_model_units": 600,
		"compute_profile": "COMPUTE_PROFILE_LOW_LATENCY"
	}]}}`, endpoints.rawWrites[0])
}

func TestServeALoggedModelOnCPU(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, loggedMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig, "watch", "true"))
	assert.Equal(t, "CPU", endpoints.endpoint.Config.ServedEntities[0].WorkloadType)
}

func TestServeAnAudioCommandReportsItsAudioRoute(t *testing.T) {
	srv, _ := serveServer(t, newFakeEndpoints(t, nil), replaceFirst("task: llm/v1/chat", "task: llm/v1/audio/transcriptions")(commandMLmodel))

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig, "watch", "true"))
	assert.Equal(t, srv.URL+"/serving-endpoints/qwen-chat/audio/transcriptions", serveJSON(t, &buf).EndpointURL)
}

func TestServeAGivenVersionDoesntListVersions(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, fake := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputText, &buf, srv.URL, gpuServeConfig, "watch", "true"))
	assert.Zero(t, fake.listings)
	assert.Equal(t, "3", endpoints.endpoint.Config.ServedEntities[0].EntityVersion)
	assert.Equal(t, 8, endpoints.endpoint.Config.ServedEntities[0].MaxProvisionedConcurrency)
	assert.NotContains(t, buf.String(), "newest READY")
}

func TestServeFromFlags(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	require.NoError(t, runModelCmd(t, newServeCommand(), flags.OutputJSON, &buf, srv.URL,
		"uc-model-name", "main.models.qwen", "endpoint-name", "from-flags", "model-version", "2", "watch", "true"))
	got := serveJSON(t, &buf)
	assert.Equal(t, "from-flags", got.Endpoint)
	assert.Equal(t, "2", got.ModelVersion)
}

func TestServeRefusesComputeOfTheOtherKindBeforeChangingTheEndpoint(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, dedicatedMLmodel)

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputJSON, &buf, srv.URL, gpuServeConfig)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "INVALID_ARGS", got.Error.Code)
	assert.Contains(t, got.Error.Message, "compute.gpu can't be set: version 3 of main.models.qwen is a checkpoint registered without engine")
	assert.Empty(t, endpoints.writes)
}

func TestServeWithoutAReadyVersionFails(t *testing.T) {
	srv, fake := serveServer(t, newFakeEndpoints(t, nil), commandMLmodel)
	fake.versions = `{"model_versions": [{"version": 5, "status": "PENDING_REGISTRATION"}]}`

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "INVALID_ARGS", got.Error.Code)
	assert.Contains(t, got.Error.Message, "main.models.qwen has no READY version to serve; register one with databricks air register, or set model_version")
}

func TestServeAVersionWithoutAnMLmodelFails(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	srv, _ := serveServer(t, endpoints, "")

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputText, &buf, srv.URL, minimalServeConfig)
	assert.ErrorContains(t, err, "version 4 of main.models.qwen doesn't exist or has no MLmodel, so air serve can't tell what serves it")
	assert.Empty(t, endpoints.writes)
}

func TestServeReportsAnEndpointThatIsNotReady(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	endpoints.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputJSON, &buf, srv.URL, minimalServeConfig, "watch", "true")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	// Running air serve again doesn't make the endpoint ready, so nothing says to.
	assert.Equal(t, "ENDPOINT_NOT_READY", got.Error.Code)
	assert.False(t, got.Error.Retryable)
	assert.NotContains(t, got.Error.Message, "run:")
}

func TestServeEndpointFailureSaysHowToRetry(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)
	endpoints.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed}
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputText, &buf, srv.URL, minimalServeConfig, "watch", "true")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serving version 4 of main.models.qwen on endpoint qwen-chat failed: ")
	assert.Contains(t, err.Error(), "To try again, run:\n  databricks air serve -f ")
	assert.Contains(t, err.Error(), " --model-version 4 --watch")
}

func TestServeRefusesToReplaceAnotherModel(t *testing.T) {
	other := testEntity("5")
	other.EntityName = "main.models.llama"
	endpoints := newFakeEndpoints(t, endpointServing(other, readyState))
	srv, _ := serveServer(t, endpoints, commandMLmodel)

	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputText, &buf, srv.URL, minimalServeConfig)
	assert.ErrorContains(t, err, "endpoint qwen-chat serves version 5 of main.models.llama")
	// Running the same command again would fail the same way, so none is suggested.
	assert.NotContains(t, err.Error(), "run:")
	assert.Empty(t, endpoints.writes)
}

func TestServeRefusesAnEmptyVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	err := runModelCmd(t, newServeCommand(), flags.OutputText, &buf, "", "uc-model-name", "main.models.qwen", "endpoint-name", "ep", "model-version", "", "dry-run", "true")
	assert.ErrorContains(t, err, "--model-version cannot be empty")
}

func TestServeDryRunNeedsNoWorkspace(t *testing.T) {
	cmd := newServeCommand()
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	assert.NoError(t, cmd.PreRunE(cmd, nil))

	var buf bytes.Buffer
	require.NoError(t, runServeCmd(t, flags.OutputText, &buf, "", minimalServeConfig, "dry-run", "true"))
	assert.Contains(t, buf.String(), "Dry run: would serve the newest READY version of main.models.qwen on endpoint qwen-chat, on the compute it was registered for.")
	assert.Contains(t, buf.String(), `Dry run: configuration for endpoint "qwen-chat" is valid; nothing was changed.`)

	buf.Reset()
	require.NoError(t, runServeCmd(t, flags.OutputJSON, &buf, "", gpuServeConfig, "dry-run", "true"))
	got := serveJSON(t, &buf)
	assert.Equal(t, "DRY_RUN_OK", got.Status)
	assert.Equal(t, &servePlan{UCModel: "main.models.qwen", Endpoint: "qwen-chat", ModelVersion: "3"}, got.Plan)
}

func TestServeFlagValidation(t *testing.T) {
	var buf bytes.Buffer
	err := runServeCmd(t, flags.OutputText, &buf, "", minimalServeConfig, "timeout", "0s", "dry-run", "true")
	assert.ErrorContains(t, err, "--timeout must be positive")
}

func TestServeResumeCommand(t *testing.T) {
	abs, err := filepath.Abs("my serve.yaml")
	require.NoError(t, err)
	opts := serveOptions{file: "my serve.yaml", overrides: []string{"compute.gpu.max_replicas=4"}, endpoint: "other"}
	assert.Equal(t,
		"databricks air serve -f "+shellquote.BashArg(filepath.ToSlash(abs))+" --override 'compute.gpu.max_replicas=4' --endpoint-name other --model-version 4 --watch -p dev",
		serveResumeCommand("dev", opts, "4"))

	assert.Equal(t,
		"databricks air serve --uc-model-name a.b.c --endpoint-name ep --model-version 2 --watch",
		serveResumeCommand("", serveOptions{ucModel: "a.b.c", endpoint: "ep"}, "2"))
}

func TestServeHelpDocumentsAField(t *testing.T) {
	var out strings.Builder
	cmd := newServeCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config.compute"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "config.compute\n  Compute to serve on")
	assert.Contains(t, out.String(), "model_units")

	out.Reset()
	cmd = newServeCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config"})
	require.NoError(t, cmd.Execute())
	// The coding agent skill writes run YAML, so the serve help leaves it out.
	assert.NotContains(t, out.String(), "databricks-ai-runtime")
}

func TestEndpointQueryURL(t *testing.T) {
	tests := []struct {
		task string
		want string
	}{
		{"llm/v1/chat", "https://h/serving-endpoints/ep/invocations"},
		{taskAudioTranscriptions, "https://h/serving-endpoints/ep/audio/transcriptions"},
		{taskAudioTranslations, "https://h/serving-endpoints/ep/audio/translations"},
		{"", "https://h/serving-endpoints/ep/invocations"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, endpointQueryURL("https://h/", "ep", tt.task))
	}
}
