package aircmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const servingEndpointsPath = "/api/2.0/serving-endpoints"

// fakeEndpoints is a stateful fake of the serving endpoints API for one endpoint.
// A create or config update is applied at once, and the endpoint then reports
// finalState (READY and NOT_UPDATING unless a test sets otherwise).
type fakeEndpoints struct {
	t          *testing.T
	mu         sync.Mutex
	endpoint   *serving.ServingEndpointDetailed
	finalState serving.EndpointState
	writes     []string
	// rawWrites holds the body of every create and config update, as sent.
	rawWrites []string
	// gets counts the GETs of an existing endpoint. Those numbered in failOnGets
	// answer 500, and from readyAfterGets on (when set) the endpoint is ready.
	gets           int
	failOnGets     map[int]bool
	readyAfterGets int
}

func newFakeEndpoints(t *testing.T, existing *serving.ServingEndpointDetailed) *fakeEndpoints {
	return &fakeEndpoints{
		t:          t,
		endpoint:   existing,
		finalState: serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating},
	}
}

func (f *fakeEndpoints) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodPost && r.URL.Path == servingEndpointsPath:
		var req serving.CreateServingEndpoint
		if !f.decode(w, r, &req) {
			return
		}
		f.writes = append(f.writes, "create")
		f.endpoint = &serving.ServingEndpointDetailed{Name: req.Name, Config: f.output(req.Config.ServedEntities)}
		f.endpoint.State = &f.finalState
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/config"):
		var req serving.EndpointCoreConfigInput
		if !f.decode(w, r, &req) {
			return
		}
		if updating(f.endpoint) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error_code": "RESOURCE_CONFLICT", "message": "The endpoint is being updated."}`))
			return
		}
		f.writes = append(f.writes, "update")
		f.endpoint.Config = f.output(req.ServedEntities)
		f.endpoint.PendingConfig = nil
		f.endpoint.State = &f.finalState
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, servingEndpointsPath+"/"):
		if f.endpoint != nil {
			f.gets++
			if f.failOnGets[f.gets] {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_code": "INTERNAL_ERROR", "message": "Something went wrong."}`))
				return
			}
			if f.readyAfterGets > 0 && f.gets >= f.readyAfterGets {
				f.endpoint.State = &readyState
			}
		}
		if f.endpoint == nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Endpoint not found"}`))
			return
		}
	default:
		_, _ = w.Write([]byte(`{}`))
		return
	}
	assert.NoError(f.t, json.NewEncoder(w).Encode(f.endpoint))
}

// decode reads a request body into v, failing the request (not the handler
// goroutine) when it can't.
func (f *fakeEndpoints) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(r.Body)
	if !assert.NoError(f.t, err) || !assert.NoError(f.t, json.Unmarshal(raw, v)) {
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	f.rawWrites = append(f.rawWrites, string(raw))
	return true
}

func (f *fakeEndpoints) output(entities []serving.ServedEntityInput) *serving.EndpointCoreConfigOutput {
	out := &serving.EndpointCoreConfigOutput{ConfigVersion: 1}
	for _, e := range entities {
		out.ServedEntities = append(out.ServedEntities, servedOutput(e))
	}
	return out
}

func servedOutput(e serving.ServedEntityInput) serving.ServedEntityOutput {
	return serving.ServedEntityOutput{
		EntityName:         e.EntityName,
		EntityVersion:      e.EntityVersion,
		WorkloadType:       e.WorkloadType,
		WorkloadSize:       e.WorkloadSize,
		ScaleToZeroEnabled: e.ScaleToZeroEnabled,
		EnvironmentVars:    e.EnvironmentVars,
	}
}

func (f *fakeEndpoints) start() *httptest.Server {
	srv := httptest.NewServer(f)
	f.t.Cleanup(srv.Close)
	return srv
}

func testEntity(version string) serving.ServedEntityInput {
	return serving.ServedEntityInput{
		EntityName:      "main.models.qwen",
		EntityVersion:   version,
		WorkloadType:    serving.ServingModelWorkloadTypeGpuMedium,
		WorkloadSize:    "Small",
		EnvironmentVars: map[string]string{"HF_TOKEN": "{{secrets/llm/hf_token}}"},
	}
}

func endpointServing(entity serving.ServedEntityInput, state serving.EndpointState) *serving.ServingEndpointDetailed {
	return &serving.ServingEndpointDetailed{
		Name:   "qwen-chat",
		Config: &serving.EndpointCoreConfigOutput{ServedEntities: []serving.ServedEntityOutput{servedOutput(entity)}},
		State:  &state,
	}
}

var readyState = serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}

func TestServeOnEndpointCreatesAMissingEndpoint(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	srv := fake.start()

	endpoint, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"create"}, fake.writes)
	assert.Equal(t, "3", endpoint.Config.ServedEntities[0].EntityVersion)
}

func TestServeOnEndpointUpdatesAnotherVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("2"), readyState))
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
}

func TestServeOnEndpointUpdatesChangedCompute(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), readyState))
	srv := fake.start()
	entity := testEntity("3")
	entity.WorkloadSize = "Medium"

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", entity, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
}

func TestServeOnEndpointLeavesAServedVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), readyState))
	srv := fake.start()
	// Unity Catalog reports names lowercased.
	entity := testEntity("3")
	entity.EntityName = "Main.Models.Qwen"

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", entity, nil)
	require.NoError(t, err)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointWaitsForItsOwnPendingUpdate(t *testing.T) {
	existing := endpointServing(testEntity("2"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{servedOutput(testEntity("3"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	polls := 0
	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), func(*serving.ServingEndpointDetailed) {
		// The pending update finishes after the first poll.
		polls++
		fake.mu.Lock()
		defer fake.mu.Unlock()
		fake.endpoint = endpointServing(testEntity("3"), readyState)
	})
	require.NoError(t, err)
	assert.Empty(t, fake.writes)
	assert.Positive(t, polls)
}

func TestServeOnEndpointWaitsForAnotherUpdateBeforeUpdating(t *testing.T) {
	existing := endpointServing(testEntity("1"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{servedOutput(testEntity("2"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), func(endpoint *serving.ServingEndpointDetailed) {
		// The other update finishes after the first poll.
		if updating(endpoint) {
			fake.mu.Lock()
			defer fake.mu.Unlock()
			fake.endpoint = endpointServing(testEntity("2"), readyState)
		}
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
	assert.Equal(t, "3", fake.endpoint.Config.ServedEntities[0].EntityVersion)
}

func TestServeOnEndpointRefusesToReplaceOtherEntities(t *testing.T) {
	other := testEntity("5")
	other.EntityName = "main.models.llama"
	split := endpointServing(testEntity("2"), readyState)
	split.Config.ServedEntities = append(split.Config.ServedEntities, servedOutput(testEntity("1")))
	tests := []struct {
		name     string
		existing *serving.ServingEndpointDetailed
		wantErr  string
	}{
		{"another model", endpointServing(other, readyState), "endpoint qwen-chat serves version 5 of main.models.llama, and air deploy replaces only a single version of main.models.qwen"},
		{"a traffic split", split, "endpoint qwen-chat serves version 2 of main.models.qwen and version 1 of main.models.qwen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEndpoints(t, tt.existing)
			w := newTestWorkspaceClient(t, fake.start().URL)

			assert.ErrorContains(t, checkEndpoint(withDeadline(t), w, "qwen-chat", "main.models.qwen"), tt.wantErr)
			_, err := serveOnEndpoint(withDeadline(t), w, "qwen-chat", testEntity("3"), nil)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, fake.writes)
		})
	}
}

func TestCheckEndpointAllowsAMissingEndpoint(t *testing.T) {
	srv := newFakeEndpoints(t, nil).start()
	assert.NoError(t, checkEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", "main.models.qwen"))
}

func TestServeOnEndpointWaitsThroughTransientFailures(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	// The endpoint starts over three polls, and the first GET of each fails: more
	// failures than one request may retry, spread over the wait.
	fake.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress}
	fake.failOnGets = map[int]bool{1: true, 3: true, 5: true}
	fake.readyAfterGets = 6
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"create"}, fake.writes)
	assert.Equal(t, 6, fake.gets)
}

func TestServeOnEndpointReportsAFailedUpdate(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	fake.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed}
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), nil)
	assert.ErrorContains(t, err, "UPDATE_FAILED")
}

func TestServeOnEndpointReportsAnEndpointThatIsNotReady(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}))
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), nil)
	assert.ErrorContains(t, err, "endpoint qwen-chat serves version 3 of main.models.qwen but isn't ready")
}

func TestServesOnly(t *testing.T) {
	entity := testEntity("3")
	assert.True(t, servesOnly([]serving.ServedEntityOutput{servedOutput(entity)}, entity))
	assert.False(t, servesOnly(nil, entity))
	assert.False(t, servesOnly([]serving.ServedEntityOutput{servedOutput(entity), servedOutput(testEntity("2"))}, entity))

	changed := entity
	changed.EnvironmentVars = map[string]string{"HF_TOKEN": "{{secrets/llm/other}}"}
	assert.False(t, servesOnly([]serving.ServedEntityOutput{servedOutput(entity)}, changed))

	scaled := entity
	scaled.ScaleToZeroEnabled = true
	assert.False(t, servesOnly([]serving.ServedEntityOutput{servedOutput(entity)}, scaled))
}
