package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestPreflightValidateFailsOpenWhenBackendUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"disabled", http.StatusBadRequest, `{"error_code":"FEATURE_DISABLED","message":"not enabled"}`},
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

func TestPreflightValidateBlocksOnCallerError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := validateServer(t, status, `{"error_code":"INVALID_PARAMETER_VALUE","message":"request rejected"}`, nil)
			err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
			require.Error(t, err)
		})
	}
}

func TestPreflightValidateBlocksOnRequestVisitorError(t *testing.T) {
	srv := validateServer(t, http.StatusOK, `{}`, nil)
	w := validationTestWorkspaceClient(t, srv.URL)
	w.Config.Headers = func(*http.Request) error {
		return errors.New("failed to add request headers")
	}

	err := preflightValidate(t.Context(), w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.ErrorContains(t, err, "failed to add request headers")
}

func TestClassifyValidationFailure(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantUnavailable bool
		wantRetryable   bool
	}{
		{"canceled", context.Canceled, false, false},
		{"deadline", context.DeadlineExceeded, true, true},
		{"transport", &url.Error{Op: "Post", URL: "https://example.test", Err: errors.New("connection failed")}, true, true},
		{"unknown", errors.New("authentication visitor failed"), false, false},
		{"bad request", &apierr.APIError{StatusCode: http.StatusBadRequest}, false, false},
		{"disabled", &apierr.APIError{StatusCode: http.StatusBadRequest, ErrorCode: "FEATURE_DISABLED"}, true, false},
		{"unauthenticated", &apierr.APIError{StatusCode: http.StatusUnauthorized}, false, false},
		{"forbidden", &apierr.APIError{StatusCode: http.StatusForbidden}, false, false},
		{"not found", &apierr.APIError{StatusCode: http.StatusNotFound}, true, false},
		{"not implemented", &apierr.APIError{StatusCode: http.StatusNotImplemented}, true, false},
		{"request timeout", &apierr.APIError{StatusCode: http.StatusRequestTimeout}, true, true},
		{"rate limited", &apierr.APIError{StatusCode: http.StatusTooManyRequests}, true, true},
		{"server error", &apierr.APIError{StatusCode: http.StatusInternalServerError}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unavailable, retryable := classifyValidationFailure(tt.err)
			assert.Equal(t, tt.wantUnavailable, unavailable)
			assert.Equal(t, tt.wantRetryable, retryable)
		})
	}
}

func TestValidateConfigRequestShape(t *testing.T) {
	var gotReq map[string]any
	srv := validateServer(t, http.StatusOK, `{}`, &gotReq)

	cfg := baseRunConfig()
	cfg.MaxRetries = new(3)
	cfg.EnvVariables = map[string]string{"HF_HOME": "/tmp/hf"}
	cfg.Environment = &environmentConfig{UnityCatalogImage: "main.air.training:v1"}
	err := preflightValidate(t.Context(), validationTestWorkspaceClient(t, srv.URL), cfg, "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)

	task := gotReq["task"].(map[string]any)
	assert.Equal(t, "llama-fine-tune", task["experiment"])
	deployment := task["deployments"].([]any)[0].(map[string]any)
	compute := deployment["compute"].(map[string]any)
	assert.Equal(t, "GPU_8xH100", compute["accelerator_type"])
	assert.EqualValues(t, 16, compute["accelerator_count"])
	assert.Equal(t, "main.air.training:v1", task["unity_catalog_image_path"])

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
