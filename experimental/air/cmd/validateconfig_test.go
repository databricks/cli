package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseRunConfig() *runConfig {
	return &runConfig{
		ExperimentName: "llama-fine-tune",
		Compute:        &computeConfig{NumAccelerators: 16, AcceleratorType: "GPU_8xH100"},
	}
}

// validateServer serves one ValidateConfig response with the given status and
// body, and records the request body it received.
func validateServer(t *testing.T, status int, body string, gotReq *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == validateConfigPath {
			if gotReq != nil {
				_ = json.NewDecoder(r.Body).Decode(gotReq)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPreflightValidatePasses(t *testing.T) {
	srv := validateServer(t, http.StatusOK, `{}`, nil)
	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), baseRunConfig())
	assert.NoError(t, err)
}

func TestPreflightValidateReportsErrors(t *testing.T) {
	body := `{"errors":[
		{"path":"experiment","message":"only letters, digits, hyphens, underscores","code":"DISALLOWED_CHARACTERS"},
		{"path":"deployments[0].compute.accelerator_count","message":"must be a multiple of 8","code":"COUNT_NOT_MULTIPLE"}
	]}`
	srv := validateServer(t, http.StatusOK, body, nil)
	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), baseRunConfig())
	require.Error(t, err)
	// Every problem is surfaced, each pointing at its config field.
	assert.Contains(t, err.Error(), "experiment: only letters")
	assert.Contains(t, err.Error(), "deployments[0].compute.accelerator_count: must be a multiple of 8")
}

func TestPreflightValidateFailsOpenWhenDisabled(t *testing.T) {
	// The endpoint is behind a SAFE flag; a disabled endpoint must not block the run.
	srv := validateServer(t, http.StatusBadRequest,
		`{"error_code":"FEATURE_DISABLED","message":"ValidateConfig is not yet enabled."}`, nil)
	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), baseRunConfig())
	assert.NoError(t, err)
}

func TestPreflightValidateFailsOpenWhenNotFound(t *testing.T) {
	// A workspace that predates the endpoint returns 404; skip and let submit proceed.
	srv := validateServer(t, http.StatusNotFound, `{"error_code":"ENDPOINT_NOT_FOUND","message":"not found"}`, nil)
	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), baseRunConfig())
	assert.NoError(t, err)
}

func TestValidateConfigRequestShape(t *testing.T) {
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	cfg := baseRunConfig()
	cfg.MaxRetries = new(3)
	cfg.EnvVariables = map[string]string{"HF_HOME": "/tmp/hf"}
	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), cfg)
	require.NoError(t, err)

	task := gotReq["task"].(map[string]any)
	assert.Equal(t, "llama-fine-tune", task["experiment"])
	deployment := task["deployments"].([]any)[0].(map[string]any)
	compute := deployment["compute"].(map[string]any)
	assert.Equal(t, "GPU_8xH100", compute["accelerator_type"])
	assert.EqualValues(t, 16, compute["accelerator_count"])

	runOptions := gotReq["run_options"].(map[string]any)
	assert.EqualValues(t, 3, runOptions["max_retries"])
	assert.Equal(t, map[string]any{"HF_HOME": "/tmp/hf"}, runOptions["env_variables"])
}

func TestValidateConfigRequestOmitsUnsetOptions(t *testing.T) {
	// A minimal config carries no run_options and only the fields it set, so the
	// server never validates values the user didn't provide.
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	err := preflightValidate(t.Context(), newTestWorkspaceClient(t, srv.URL), baseRunConfig())
	require.NoError(t, err)

	_, hasRunOptions := gotReq["run_options"]
	assert.False(t, hasRunOptions)
	task := gotReq["task"].(map[string]any)
	_, hasMlflowRun := task["mlflow_run"]
	assert.False(t, hasMlflowRun)
}
