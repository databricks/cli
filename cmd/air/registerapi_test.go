package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Canned fake responses that stand for an error rather than a body.
const (
	conflictResponse       = "conflict"
	serverErrorResponse    = "server error"
	notFinalizableResponse = "not finalizable"
)

// fakeModelServing serves registerModelPackage with responses, the staging status
// API with statuses, and finalizeModelPackage with finalizations, each in turn and
// then repeating the last; statuses default to SUCCEEDED, and finalizations to the
// finalized version. It records every request. Other paths are served by endpoints
// when set, else get an empty object (the SDK's config probe).
type fakeModelServing struct {
	t                *testing.T
	mu               sync.Mutex
	responses        []string
	statuses         []string
	finalizations    []string
	bodies           []registerModelPackageRequest
	statusRequests   []stagingStatusRequest
	finalizeRequests []finalizeModelPackageRequest
	endpoints        http.HandlerFunc
}

func newFakeModelServing(t *testing.T, responses ...string) *fakeModelServing {
	return &fakeModelServing{t: t, responses: responses}
}

func (f *fakeModelServing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case registerModelPackagePath:
		var body registerModelPackageRequest
		if !f.decode(w, r, &body) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.bodies = append(f.bodies, body)
		f.respond(w, f.responses, len(f.bodies))
	case stagingStatusPath:
		var body stagingStatusRequest
		if !f.decode(w, r, &body) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.statusRequests = append(f.statusRequests, body)
		statuses := f.statuses
		if len(statuses) == 0 {
			statuses = []string{statusSucceeded}
		}
		f.respond(w, statuses, len(f.statusRequests))
	case finalizeModelPackagePath:
		var body finalizeModelPackageRequest
		if !f.decode(w, r, &body) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.finalizeRequests = append(f.finalizeRequests, body)
		finalizations := f.finalizations
		if len(finalizations) == 0 {
			finalizations = []string{finalizedResponse}
		}
		f.respond(w, finalizations, len(f.finalizeRequests))
	default:
		if f.endpoints != nil {
			f.endpoints(w, r)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}
}

// decode reads a request body into v, failing the request (not the handler
// goroutine) when it can't.
func (f *fakeModelServing) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(r.Body)
	if !assert.NoError(f.t, err) || !assert.NoError(f.t, json.Unmarshal(raw, v)) {
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	return true
}

// respond writes the nth (1-based) of responses, repeating the last.
func (f *fakeModelServing) respond(w http.ResponseWriter, responses []string, n int) {
	switch body := responses[min(n, len(responses))-1]; body {
	case conflictResponse:
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error_code": "RESOURCE_CONFLICT", "message": "The idempotency key can't be used for this registration. Use a new key."}`))
	case serverErrorResponse:
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code": "INTERNAL_ERROR", "message": "Something went wrong."}`))
	case notFinalizableResponse:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code": "BAD_REQUEST", "message": "Version 3 of model main.models.qwen can't be finalized yet: the checkpoint's weights haven't been copied into it."}`))
	default:
		_, _ = w.Write([]byte(body))
	}
}

func (f *fakeModelServing) start() *httptest.Server {
	srv := httptest.NewServer(f)
	f.t.Cleanup(srv.Close)
	return srv
}

func (f *fakeModelServing) requests() []registerModelPackageRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]registerModelPackageRequest(nil), f.bodies...)
}

const (
	stagingInProgress = `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3", "ready": false,
		"staging_status": {"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	stagingSucceeded = `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3", "ready": false,
		"staging_status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	stagingReady = `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3", "ready": true,
		"staging_status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	stagingFailed = `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3", "ready": false,
		"staging_status": {"state": "FAILED", "model_artifact": {"state": "FAILED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`

	statusInProgress = `{"status": {"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	statusSucceeded  = `{"status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	statusFailed     = `{"status": {"state": "FAILED", "model_artifact": {"state": "FAILED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`

	finalizedResponse = `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3",
		"staging_status": {"state": "IN_PROGRESS", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "IN_PROGRESS"}}}`
)

// fastRetries shortens the pause before air register repeats a failed request.
func fastRetries(ctx context.Context) context.Context {
	return env.Set(ctx, retry.IntervalMsVariable, "1")
}

func checkpointRequest() registerModelPackageRequest {
	return registerModelPackageRequest{
		ModelSource:    "/Volumes/main/models/weights/qwen3_8b",
		UCModel:        "main.models.qwen",
		IdempotencyKey: "key-1",
		Engine: &modelPackageEngine{
			Environment: modelPackageEnvironment{Version: "6"},
			Command:     "python -m vllm.entrypoints.openai.api_server --model qwen3_8b",
			Task:        "llm/v1/chat",
		},
	}
}

func registerTestCheckpoint(t *testing.T, fake *fakeModelServing, req registerModelPackageRequest) (*registerModelPackageResponse, []string, error) {
	t.Helper()
	srv := fake.start()
	var states []string
	resp, err := registerAndFinalize(fastRetries(t.Context()), newTestWorkspaceClient(t, srv.URL), req, time.Millisecond, func(r *registerModelPackageResponse) {
		states = append(states, r.weightsState())
	})
	return resp, states, err
}

func TestRegisterAndFinalizePollsStagingThenFinalizes(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress)
	fake.statuses = []string{statusInProgress, statusSucceeded}
	req := checkpointRequest()
	req.RetryFailedStaging = true

	resp, states, err := registerTestCheckpoint(t, fake, req)
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Equal(t, "3", resp.ModelVersion)
	assert.Equal(t, []string{"IN_PROGRESS", "IN_PROGRESS", "SUCCEEDED"}, states)

	// One registration; the status is polled for the registered version until the
	// weights are copied, and the version is then finalized.
	assert.Equal(t, []registerModelPackageRequest{req}, fake.requests())
	assert.Equal(t, []stagingStatusRequest{{Model: "main.models.qwen", ModelVersion: "3"}, {Model: "main.models.qwen", ModelVersion: "3"}}, fake.statusRequests)
	assert.Equal(t, []finalizeModelPackageRequest{{UCModel: "main.models.qwen", ModelVersion: "3"}}, fake.finalizeRequests)
}

func TestRegisterAndFinalizeReportsTheFinalizedStagingStatus(t *testing.T) {
	fake := newFakeModelServing(t, stagingSucceeded)

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.NoError(t, err)
	// The environment is still being staged when the version is finalized.
	assert.Equal(t, stagingStateInProgress, resp.environmentState())
	assert.Empty(t, fake.statusRequests)
}

func TestRegisterAndFinalizeReturnsAReadyVersion(t *testing.T) {
	fake := newFakeModelServing(t, stagingReady)

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Empty(t, fake.statusRequests)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterAndFinalizeReportsFailedStaging(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress)
	fake.statuses = []string{statusFailed}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.ErrorIs(t, err, errStagingFailed)
	assert.Equal(t, "3", resp.ModelVersion)
	assert.Len(t, fake.requests(), 1)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterAndFinalizeReportsAFailureTheRegistrationReports(t *testing.T) {
	fake := newFakeModelServing(t, stagingFailed)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.ErrorIs(t, err, errStagingFailed)
	assert.Empty(t, fake.statusRequests)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterAndFinalizeReturnsWhyTheServerWontFinalize(t *testing.T) {
	notStaged := strings.ReplaceAll(stagingInProgress, "IN_PROGRESS", "NOT_STAGED")
	fake := newFakeModelServing(t, notStaged)
	fake.finalizations = []string{notFinalizableResponse}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	_, ok := errors.AsType[*finalizeError](err)
	assert.True(t, ok)
	assert.ErrorContains(t, err, "haven't been copied into it")
	assert.False(t, resp.Ready)
	// A version with no staging record isn't polled; the server says what to do.
	assert.Empty(t, fake.statusRequests)
	assert.Len(t, fake.finalizeRequests, 1)
}

// Staging statuses that break the contract: the server always reports a known state
// for both the weights and the serving environment.
var malformedStagingStatuses = map[string]struct {
	status  string
	wantErr string
}{
	"no status":                       {`null`, "it has no staging status"},
	"no weights":                      {`{"state": "IN_PROGRESS", "environment_artifact": {"state": "SUCCEEDED"}}`, "its staging status has no state for the weights"},
	"weights without a state":         {`{"state": "IN_PROGRESS", "model_artifact": {}, "environment_artifact": {"state": "SUCCEEDED"}}`, "its staging status has no state for the weights"},
	"unknown weights state":           {`{"state": "IN_PROGRESS", "model_artifact": {"state": "COPYING"}, "environment_artifact": {"state": "SUCCEEDED"}}`, `unknown state "COPYING" for the weights`},
	"no environment":                  {`{"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}}`, "its staging status has no state for the serving environment"},
	"unknown environment state":       {`{"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}, "environment_artifact": {"state": "BUILDING"}}`, `unknown state "BUILDING" for the serving environment`},
	"control characters in the state": {`{"state": "IN_PROGRESS", "model_artifact": {"state": "X\u001b[31m"}, "environment_artifact": {"state": "SUCCEEDED"}}`, `unknown state "X" for the weights`},
}

func TestRegisterAndFinalizeRejectsAMalformedRegistrationStatus(t *testing.T) {
	for name, tt := range malformedStagingStatuses {
		t.Run(name, func(t *testing.T) {
			fake := newFakeModelServing(t, `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3", "ready": false, "staging_status": `+tt.status+`}`)

			resp, states, err := registerTestCheckpoint(t, fake, checkpointRequest())
			assert.ErrorContains(t, err, "air register can't read Model Serving's response about version 3 of main.models.qwen: ")
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, resp)
			assert.Empty(t, states)
			assert.Empty(t, fake.statusRequests)
			assert.Empty(t, fake.finalizeRequests)
		})
	}
}

func TestRegisterAndFinalizeRejectsAMalformedPolledStatus(t *testing.T) {
	for name, tt := range malformedStagingStatuses {
		t.Run(name, func(t *testing.T) {
			fake := newFakeModelServing(t, stagingInProgress)
			fake.statuses = []string{`{"status": ` + tt.status + `}`, statusSucceeded}

			resp, states, err := registerTestCheckpoint(t, fake, checkpointRequest())
			assert.ErrorContains(t, err, tt.wantErr)
			code, retryable := modelErrorCode(err)
			assert.Equal(t, codeInternalError, code)
			assert.True(t, retryable)
			// It stops at the malformed status instead of reading it as progress.
			assert.Equal(t, []string{"IN_PROGRESS"}, states)
			assert.Len(t, fake.statusRequests, 1)
			assert.Empty(t, fake.finalizeRequests)
			assert.False(t, resp.Ready)
		})
	}
}

func TestRegisterAndFinalizeRejectsAMalformedFinalizedStatus(t *testing.T) {
	for name, tt := range malformedStagingStatuses {
		t.Run(name, func(t *testing.T) {
			fake := newFakeModelServing(t, stagingSucceeded)
			fake.finalizations = []string{`{"model_version": "3", "staging_status": ` + tt.status + `}`}

			resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
			_, ok := errors.AsType[*finalizeError](err)
			assert.True(t, ok)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.False(t, resp.Ready)
		})
	}
}

func TestRegisterAndFinalizeUsesTheFinalizedStatusAsItIs(t *testing.T) {
	environmentFailed := strings.Replace(stagingSucceeded, `"environment_artifact": {"state": "SUCCEEDED"}`, `"environment_artifact": {"state": "FAILED"}`, 1)
	fake := newFakeModelServing(t, environmentFailed)
	fake.finalizations = []string{`{"model_version": "3", "staging_status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.NoError(t, err)
	// The finalize response's status replaces the registration's, not merged with it.
	assert.Equal(t, stagingStateSucceeded, resp.environmentState())
}

func TestRegisterAndFinalizeRejectsAnotherFinalizedVersion(t *testing.T) {
	fake := newFakeModelServing(t, stagingSucceeded)
	fake.finalizations = []string{`{"model_version": "4"}`}

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	_, ok := errors.AsType[*finalizeError](err)
	assert.True(t, ok)
	assert.ErrorContains(t, err, `the server finalized version "4" instead`)
}

func TestRegisterAndFinalizeRejectsARegistrationWithoutAVersion(t *testing.T) {
	fake := newFakeModelServing(t, `{"model_id": "m-1", "uc_model": "main.models.qwen", "ready": false}`)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, "registering a version of main.models.qwen returned no version")
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterAndFinalizeRejectsAVersionThatIsntANumber(t *testing.T) {
	fake := newFakeModelServing(t, `{"model_id": "m-1", "uc_model": "main.models.qwen", "model_version": "3; rm", "ready": false, "staging_status": {"model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, `registering a version of main.models.qwen returned "3; rm", which isn't a version number`)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegisterAndFinalizeRetriesTransientErrors(t *testing.T) {
	fake := newFakeModelServing(t, serverErrorResponse, stagingInProgress)
	fake.statuses = []string{serverErrorResponse, statusSucceeded}
	fake.finalizations = []string{serverErrorResponse, finalizedResponse}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Len(t, fake.requests(), 2)
	assert.Len(t, fake.statusRequests, 2)
	assert.Len(t, fake.finalizeRequests, 2)
}

func TestRegisterAndFinalizeGivesUpOnPersistentErrors(t *testing.T) {
	fake := newFakeModelServing(t, serverErrorResponse)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, "Something went wrong.")
	assert.Len(t, fake.requests(), retry.MaxRetries+1)
}

func TestRegisterAndFinalizeReturnsServerErrors(t *testing.T) {
	fake := newFakeModelServing(t, conflictResponse)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, "The idempotency key can't be used for this registration")
	assert.Len(t, fake.requests(), 1)
}

func TestRegisterAndFinalizeStopsWhenCanceled(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress)
	srv := fake.start()
	ctx, cancel := context.WithCancel(t.Context())

	_, err := registerAndFinalize(ctx, newTestWorkspaceClient(t, srv.URL), checkpointRequest(), time.Hour, func(*registerModelPackageResponse) { cancel() })
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, fake.finalizeRequests)
}

func TestRegistrationRequestForCommandJSON(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, commandRegisterConfig)
	require.NoError(t, err)
	raw, err := json.Marshal(cfg.registrationRequest("k"))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model_source": "/Volumes/main/models/weights/qwen3_8b",
		"uc_model": "main.models.qwen",
		"idempotency_key": "k",
		"engine": {
			"environment": {"version": "6", "dependencies": ["einops==0.8.0"]},
			"command": "python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080",
			"task": "llm/v1/chat"
		}
	}`, string(raw))
}

func TestRegistrationRequestForDedicatedJSON(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, dedicatedRegisterConfig)
	require.NoError(t, err)
	raw, err := json.Marshal(cfg.registrationRequest("k"))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model_source": "/Volumes/main/models/weights/qwen3_8b",
		"uc_model": "main.models.qwen",
		"idempotency_key": "k"
	}`, string(raw))
}

func TestRegisterModelPackageRequestJSON(t *testing.T) {
	raw, err := json.Marshal(checkpointRequest())
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model_source": "/Volumes/main/models/weights/qwen3_8b",
		"uc_model": "main.models.qwen",
		"idempotency_key": "key-1",
		"engine": {
			"environment": {"version": "6"},
			"command": "python -m vllm.entrypoints.openai.api_server --model qwen3_8b",
			"task": "llm/v1/chat"
		}
	}`, string(raw))
}
