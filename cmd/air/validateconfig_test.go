package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	sdkauth "github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func validationTestWorkspaceClientWithTransport(t *testing.T, transport validationRoundTripFunc) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:          "https://example.test",
		Token:         "token",
		HTTPTransport: transport,
	})
	require.NoError(t, err)
	return w
}

type validationRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn validationRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type validationOAuthCredentials struct {
	tokenSource sdkauth.TokenSource
}

func (validationOAuthCredentials) Name() string { return "oauth-m2m" }

func (c validationOAuthCredentials) Configure(context.Context, *sdkconfig.Config) (credentials.CredentialsProvider, error) {
	return credentials.NewOAuthCredentialsProviderFromTokenSource(c.tokenSource), nil
}

type validationErrorReadCloser struct{}

func (validationErrorReadCloser) Read([]byte) (int, error) { return 0, errors.New("body read failed") }
func (validationErrorReadCloser) Close() error             { return nil }

type validationContextReadCloser struct {
	ctx context.Context
}

func (r validationContextReadCloser) Read([]byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}
func (validationContextReadCloser) Close() error { return nil }

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

func TestPreflightValidationUnavailableWarnsWithoutRetry(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == validateConfigPath {
			requests.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error_code":"TEMPORARILY_UNAVAILABLE","message":"try again"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	err := preflightValidate(ctx, validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)
	assert.Equal(t, int32(1), requests.Load())
	assert.Equal(t, "Warning: server-side config validation was unavailable; continuing with submission.\n", stderr.String())
}

func TestPreflightValidationDoesNotRetryTransportFailure(t *testing.T) {
	var requests atomic.Int32
	w := validationTestWorkspaceClientWithTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == validateConfigPath {
			requests.Add(1)
		}
		return nil, errors.New("connection failed")
	})

	err := preflightValidate(t.Context(), w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.NoError(t, err)
	assert.Equal(t, int32(1), requests.Load())
}

func TestPreflightValidationFailsOpenOnInvalidSuccessResponse(t *testing.T) {
	tests := []struct {
		name string
		body func() io.ReadCloser
	}{
		{"body read failure", func() io.ReadCloser { return validationErrorReadCloser{} }},
		{"malformed JSON", func() io.ReadCloser { return io.NopCloser(strings.NewReader("{")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			w := validationTestWorkspaceClientWithTransport(t, func(req *http.Request) (*http.Response, error) {
				body := io.NopCloser(strings.NewReader(`{}`))
				if req.URL.Path == validateConfigPath {
					requests.Add(1)
					body = tt.body()
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       body,
					Request:    req,
				}, nil
			})

			err := preflightValidate(t.Context(), w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
			require.NoError(t, err)
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}

func TestPreflightValidationBodyReadFailureOnCallerErrorBlocks(t *testing.T) {
	w := validationTestWorkspaceClientWithTransport(t, func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`{}`))
		status := http.StatusOK
		if req.URL.Path == validateConfigPath {
			body = validationContextReadCloser{ctx: req.Context()}
			status = http.StatusUnauthorized
		}
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       body,
			Request:    req,
		}, nil
	})

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err := preflightValidate(ctx, w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.Error(t, err)
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
}

func TestPreflightValidationOAuthRefreshUsesDeadline(t *testing.T) {
	refreshStarted := make(chan struct{})
	tokenSource := sdkauth.NewCachedTokenSource(
		sdkauth.TokenSourceFn(func(ctx context.Context) (*oauth2.Token, error) {
			close(refreshStarted)
			<-ctx.Done()
			return nil, ctx.Err()
		}),
		sdkauth.WithCachedToken(&oauth2.Token{
			AccessToken: "expired",
			Expiry:      time.Now().Add(-time.Hour),
		}),
	)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        "https://example.test",
		Credentials: validationOAuthCredentials{tokenSource: tokenSource},
		HTTPTransport: validationRoundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("validation request should not be sent")
			return nil, nil
		}),
		HostMetadataResolver: func(context.Context, string) (*sdkconfig.HostMetadata, error) {
			return &sdkconfig.HostMetadata{}, nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = preflightValidate(ctx, w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")

	require.NoError(t, err)
	assert.Less(t, time.Since(started), time.Second)
	select {
	case <-refreshStarted:
	default:
		t.Fatal("OAuth token refresh did not start")
	}
}

func TestPreflightValidationTimeoutFailsOpen(t *testing.T) {
	releaseRequest := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == validateConfigPath {
			<-releaseRequest
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := preflightValidate(ctx, validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	close(releaseRequest)
	require.NoError(t, err)
	assert.Less(t, time.Since(started), time.Second)
}

func TestPreflightValidationPropagatesCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == validateConfigPath {
			<-r.Context().Done()
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := preflightValidate(ctx, validationTestWorkspaceClient(t, srv.URL), baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.ErrorIs(t, err, context.Canceled)
}

func TestPreflightValidationPropagatesCancellationWhileReadingResponse(t *testing.T) {
	validationStarted := make(chan struct{})
	w := validationTestWorkspaceClientWithTransport(t, func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`{}`))
		if req.URL.Path == validateConfigPath {
			close(validationStarted)
			body = validationContextReadCloser{ctx: req.Context()}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       body,
			Request:    req,
		}, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-validationStarted
		cancel()
	}()
	err := preflightValidate(ctx, w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
	require.ErrorIs(t, err, context.Canceled)
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
		{"OAuth rejection", &url.Error{Op: "Post", URL: "https://accounts.example.test", Err: &apierr.APIError{StatusCode: http.StatusUnauthorized}}, false, false},
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

func TestValidateConfigRequestCarriesEnvironment(t *testing.T) {
	cfg := baseRunConfig()
	cfg.Environment = &environmentConfig{
		Version:      stringOrInt{set: true, raw: "databricks_ai_v6"},
		Dependencies: dependencies{set: true, list: []string{"torch==2.3.0", "numpy"}},
	}

	request := validateConfigRequest(t.Context(), cfg, "/Workspace/Users/me/cmd.sh", nil, "token")
	raw, err := json.Marshal(request["environment"])
	require.NoError(t, err)
	assert.JSONEq(t, `{"base_environment":"workspace-base-environments/databricks_ai_v6","dependencies":["torch==2.3.0","numpy"]}`, string(raw))
}
