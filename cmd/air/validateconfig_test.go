package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
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

func validationTestWorkspaceClient(t *testing.T, host string) *databricks.WorkspaceClient {
	t.Helper()
	w := newTestWorkspaceClient(t, host)
	w.Config.RetryTimeoutSeconds = 1
	return w
}

func TestPreflightValidatePasses(t *testing.T) {
	srv := validateServer(t, http.StatusOK, `{}`, nil)
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	assert.NoError(t, err)
}

func TestPreflightValidateReportsErrors(t *testing.T) {
	body := `{"errors":[
		{"path":"experiment","message":"only letters, digits, hyphens, underscores","code":"DISALLOWED_CHARACTERS"},
		{"path":"deployments[0].compute.accelerator_count","message":"must be a multiple of 8","code":"COUNT_NOT_MULTIPLE"}
	]}`
	srv := validateServer(t, http.StatusOK, body, nil)
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.Error(t, err)
	// Every problem is surfaced, each pointing at its config field.
	assert.Contains(t, err.Error(), "experiment: only letters")
	assert.Contains(t, err.Error(), "deployments[0].compute.accelerator_count: must be a multiple of 8")
}

func TestPreflightValidateFailsOpenWhenBackendUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"disabled", http.StatusBadRequest, `{"error_code":"FEATURE_DISABLED","message":"not enabled"}`},
		{"not found", http.StatusNotFound, `{"error_code":"ENDPOINT_NOT_FOUND","message":"not found"}`},
		{"server error", http.StatusInternalServerError, `{"error_code":"INTERNAL_ERROR","message":"backend failed"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := validateServer(t, tt.status, tt.body, nil)
			err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
			assert.NoError(t, err)
		})
	}
}

func TestPreflightValidateBlocksOnClientError(t *testing.T) {
	// A 4xx other than the fail-open cases means the request itself was rejected
	// (e.g. the proto hook flagged a missing required field); surface it.
	srv := validateServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_PARAMETER_VALUE","message":"command_path is required"}`, nil)
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.Error(t, err)
}

func TestValidationCouldNotComplete(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, true},
		{"transport", errors.New("connection failed"), true},
		{"bad request", &apierr.APIError{StatusCode: http.StatusBadRequest}, false},
		{"disabled", &apierr.APIError{StatusCode: http.StatusBadRequest, ErrorCode: "FEATURE_DISABLED"}, true},
		{"unauthenticated", &apierr.APIError{StatusCode: http.StatusUnauthorized}, false},
		{"forbidden", &apierr.APIError{StatusCode: http.StatusForbidden}, false},
		{"not found", &apierr.APIError{StatusCode: http.StatusNotFound}, true},
		{"not implemented", &apierr.APIError{StatusCode: http.StatusNotImplemented}, true},
		{"request timeout", &apierr.APIError{StatusCode: http.StatusRequestTimeout}, true},
		{"rate limited", &apierr.APIError{StatusCode: http.StatusTooManyRequests}, true},
		{"server error", &apierr.APIError{StatusCode: http.StatusInternalServerError}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, validationCouldNotComplete(tt.err))
		})
	}
}

func TestValidateConfigRequestShape(t *testing.T) {
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	cfg := baseRunConfig()
	cfg.MaxRetries = new(3)
	cfg.EnvVariables = map[string]string{"HF_HOME": "/tmp/hf"}
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), cfg, "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)

	task := gotReq["task"].(map[string]any)
	assert.Equal(t, "llama-fine-tune", task["experiment"])
	deployment := task["deployments"].([]any)[0].(map[string]any)
	compute := deployment["compute"].(map[string]any)
	assert.Equal(t, "GPU_8xH100", compute["accelerator_type"])
	assert.EqualValues(t, 16, compute["accelerator_count"])

	runOptions := gotReq["run_options"].(map[string]any)
	assert.Equal(t, "token", runOptions["idempotency_token"])
	assert.EqualValues(t, 3, runOptions["max_retries"])
	assert.Equal(t, map[string]any{"HF_HOME": "/tmp/hf"}, runOptions["env_variables"])
}

func TestValidateConfigRequestCarriesPriorityClass(t *testing.T) {
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	cfg := baseRunConfig()
	cfg.Compute.PoolID = new("cap-8xh100-res")
	cfg.Compute.PriorityClass = new("CRITICAL")
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), cfg, "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)

	task := gotReq["task"].(map[string]any)
	// priority_class is task-level; provisioned_capacity_id stays on the compute spec.
	assert.Equal(t, "CRITICAL", task["priority_class"])
	compute := task["deployments"].([]any)[0].(map[string]any)["compute"].(map[string]any)
	assert.Equal(t, "cap-8xh100-res", compute["provisioned_capacity_id"])
}

func TestValidateConfigRequestOmitsUnsetOptions(t *testing.T) {
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"idempotency_token": "token"}, gotReq["run_options"])
	task := gotReq["task"].(map[string]any)
	_, hasMlflowRun := task["mlflow_run"]
	assert.False(t, hasMlflowRun)
}
