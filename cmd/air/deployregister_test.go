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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModelServing serves registerModelPackage with the given responses in turn,
// repeating the last, and records every registration body. Other paths are served
// by endpoints when set, else get an empty object (the SDK's config probe).
type fakeModelServing struct {
	t         *testing.T
	mu        sync.Mutex
	responses []string
	bodies    []registerModelPackageRequest
	endpoints http.HandlerFunc
}

func newFakeModelServing(t *testing.T, responses ...string) *fakeModelServing {
	return &fakeModelServing{t: t, responses: responses}
}

func (f *fakeModelServing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != registerModelPackagePath {
		if f.endpoints != nil {
			f.endpoints(w, r)
			return
		}
		_, _ = w.Write([]byte(`{}`))
		return
	}
	raw, err := io.ReadAll(r.Body)
	var body registerModelPackageRequest
	if !assert.NoError(f.t, err) || !assert.NoError(f.t, json.Unmarshal(raw, &body)) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.bodies = append(f.bodies, body)
	i := min(len(f.bodies), len(f.responses)) - 1
	if f.responses[i] == "" {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error_code": "RESOURCE_CONFLICT", "message": "The idempotency key can't be used for this registration. Use a new key."}`))
		return
	}
	_, _ = w.Write([]byte(f.responses[i]))
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
)

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

func TestRegisterUntilReadyRepeatsTheSameRequest(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress, stagingInProgress, stagingReady)
	srv := fake.start()

	var states []string
	req := checkpointRequest()
	req.RetryFailedStaging = true
	resp, err := registerUntilReady(t.Context(), newTestWorkspaceClient(t, srv.URL), req, time.Millisecond, func(r *registerModelPackageResponse) {
		states = append(states, r.weightsState())
	})
	require.NoError(t, err)
	assert.True(t, resp.Ready)
	assert.Equal(t, "3", resp.ModelVersion)
	assert.Equal(t, []string{"IN_PROGRESS", "IN_PROGRESS", "SUCCEEDED"}, states)

	bodies := fake.requests()
	require.Len(t, bodies, 3)
	assert.Equal(t, req, bodies[0])
	// Only the first request restages; the repeats continue the same registration.
	req.RetryFailedStaging = false
	assert.Equal(t, req, bodies[1])
	assert.Equal(t, req, bodies[2])
}

func TestRegisterUntilReadyReportsFailedStaging(t *testing.T) {
	fake := newFakeModelServing(t, stagingInProgress, stagingFailed)
	srv := fake.start()

	resp, err := registerUntilReady(t.Context(), newTestWorkspaceClient(t, srv.URL), checkpointRequest(), time.Millisecond, func(*registerModelPackageResponse) {})
	require.ErrorIs(t, err, errStagingFailed)
	assert.Equal(t, "3", resp.ModelVersion)
	assert.Len(t, fake.requests(), 2)
}

func TestRegisterUntilReadyReturnsServerErrors(t *testing.T) {
	srv := newFakeModelServing(t, "").start()

	_, err := registerUntilReady(t.Context(), newTestWorkspaceClient(t, srv.URL), checkpointRequest(), time.Millisecond, func(*registerModelPackageResponse) {})
	assert.ErrorContains(t, err, "The idempotency key can't be used for this registration")
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
