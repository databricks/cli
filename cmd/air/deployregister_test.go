package aircmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	conflictResponse    = "conflict"
	serverErrorResponse = "server error"
)

// fakeModelServing serves registerModelPackage with responses, and the staging
// status API with statuses, each in turn and then repeating the last; statuses
// default to SUCCEEDED. It records every request. Other paths are served by
// endpoints when set, else get an empty object (the SDK's config probe).
type fakeModelServing struct {
	t              *testing.T
	mu             sync.Mutex
	responses      []string
	statuses       []string
	bodies         []registerModelPackageRequest
	statusRequests []stagingStatusRequest
	endpoints      http.HandlerFunc
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
	stagingInProgress = `{"model_id": "m-1", "registered_model": "main.models.qwen", "model_version": "3", "ready": false,
		"staging_status": {"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	stagingReady = `{"model_id": "m-1", "registered_model": "main.models.qwen", "model_version": "3", "ready": true,
		"staging_status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	stagingFailed = `{"model_id": "m-1", "registered_model": "main.models.qwen", "model_version": "3", "ready": false,
		"staging_status": {"state": "FAILED", "model_artifact": {"state": "FAILED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`

	statusInProgress = `{"status": {"state": "IN_PROGRESS", "model_artifact": {"state": "IN_PROGRESS"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	statusSucceeded  = `{"status": {"state": "SUCCEEDED", "model_artifact": {"state": "SUCCEEDED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
	statusFailed     = `{"status": {"state": "FAILED", "model_artifact": {"state": "FAILED"}, "environment_artifact": {"state": "SUCCEEDED"}}}`
)

// fastRetries shortens the pause before air deploy repeats a failed request.
func fastRetries(ctx context.Context) context.Context {
	return env.Set(ctx, retry.IntervalMsVariable, "1")
}

func checkpointRequest() registerModelPackageRequest {
	return registerModelPackageRequest{
		RegisteredModel: "main.models.qwen",
		IdempotencyKey:  "key-1",
		Checkpoint: &modelPackageCheckpoint{
			VolumePath:     "/Volumes/main/models/weights/qwen3_8b",
			ServingCommand: "python -m vllm.entrypoints.openai.api_server --model qwen3_8b",
			Task:           "llm/v1/chat",
			Environment:    modelPackageEnvironment{Runtime: "client.6-gpu"},
		},
	}
}

func registerTestCheckpoint(t *testing.T, fake *fakeModelServing, req registerModelPackageRequest) (*registerModelPackageResponse, []string, error) {
	t.Helper()
	srv := fake.start()
	var states []string
	resp, err := registerUntilReady(fastRetries(t.Context()), newTestWorkspaceClient(t, srv.URL), req, time.Millisecond, func(r *registerModelPackageResponse) {
		states = append(states, r.weightsState())
	})
	return resp, states, err
}

func TestRegisterUntilReadyPollsStagingThenFinalizes(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress, stagingReady)
	fake.statuses = []string{statusInProgress, statusSucceeded}
	req := checkpointRequest()
	req.RetryFailedStaging = true

	resp, states, err := registerTestCheckpoint(t, fake, req)
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Equal(t, "3", resp.ModelVersion)
	assert.Equal(t, []string{"IN_PROGRESS", "IN_PROGRESS", "SUCCEEDED", "SUCCEEDED"}, states)

	// The staging status is polled for the registered version; the registration is
	// repeated once the weights are staged, which finalizes the version.
	assert.Equal(t, []stagingStatusRequest{{Model: "main.models.qwen", ModelVersion: "3"}, {Model: "main.models.qwen", ModelVersion: "3"}}, fake.statusRequests)
	bodies := fake.requests()
	require.Len(t, bodies, 2)
	assert.Equal(t, req, bodies[0])
	// Only the first request restages; the repeat continues the same registration.
	req.RetryFailedStaging = false
	assert.Equal(t, req, bodies[1])
}

func TestRegisterUntilReadyReportsFailedStaging(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress, stagingFailed)
	fake.statuses = []string{statusFailed}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.ErrorIs(t, err, errStagingFailed)
	assert.Equal(t, "3", resp.ModelVersion)
	// The failure the status reports is confirmed by the registration itself.
	assert.Len(t, fake.requests(), 2)
}

func TestRegisterUntilReadyRetriesTransientErrors(t *testing.T) {
	fake := newFakeModelServing(t, serverErrorResponse, stagingInProgress, stagingReady)
	fake.statuses = []string{serverErrorResponse, statusSucceeded}

	resp, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Len(t, fake.requests(), 3)
	assert.Len(t, fake.statusRequests, 2)
}

func TestRegisterUntilReadyDoesNotRepeatANotReadyModelFromCode(t *testing.T) {
	fake := newFakeModelServing(t, `{"model_id": "m-2", "registered_model": "a.b.c", "model_version": "1", "ready": false}`)
	req := registerModelPackageRequest{RegisteredModel: "a.b.c", IdempotencyKey: "k", Code: &modelPackageCode{EntryFile: "agent.py"}}

	_, _, err := registerTestCheckpoint(t, fake, req)
	assert.ErrorContains(t, err, "version 1 of a.b.c isn't ready after registering")
	assert.Len(t, fake.requests(), 1)
}

func TestRegisterUntilReadyGivesUpOnPersistentErrors(t *testing.T) {
	fake := newFakeModelServing(t, serverErrorResponse)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, "Something went wrong.")
	assert.Len(t, fake.requests(), retry.MaxRetries+1)
}

func TestRegisterUntilReadyReturnsServerErrors(t *testing.T) {
	fake := newFakeModelServing(t, conflictResponse)

	_, _, err := registerTestCheckpoint(t, fake, checkpointRequest())
	assert.ErrorContains(t, err, "The idempotency key can't be used for this registration")
	assert.Len(t, fake.requests(), 1)
}

func TestRegisterUntilReadyStopsWhenCanceled(t *testing.T) {
	srv := newFakeModelServing(t, stagingInProgress).start()
	ctx, cancel := context.WithCancel(t.Context())

	_, err := registerUntilReady(ctx, newTestWorkspaceClient(t, srv.URL), checkpointRequest(), time.Hour, func(*registerModelPackageResponse) { cancel() })
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRegisterModelPackageRequestJSON(t *testing.T) {
	raw, err := json.Marshal(registerModelPackageRequest{
		RegisteredModel: "a.b.c",
		IdempotencyKey:  "k",
		Code: &modelPackageCode{
			EntryFile:   "agent.py",
			EntrySource: "x = 1\n",
			Files:       []modelPackageFile{{Path: "tools/a.py", Content: ""}},
			Environment: modelPackageEnvironment{Runtime: "client.6", Dependencies: []string{"mlflow"}},
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"registered_model": "a.b.c",
		"idempotency_key": "k",
		"code": {
			"entry_file": "agent.py",
			"entry_source": "x = 1\n",
			"files": [{"path": "tools/a.py", "content": ""}],
			"environment": {"runtime": "client.6", "dependencies": ["mlflow"]}
		}
	}`, string(raw))
}
