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

// fakeEndpoints is a stateful fake of the serving endpoints API for one endpoint.
// A create or config update is applied at once, and the endpoint then reports
// finalState (READY and NOT_UPDATING unless a test sets otherwise).
type fakeEndpoints struct {
	t          *testing.T
	mu         sync.Mutex
	endpoint   *servingEndpoint
	finalState serving.EndpointState
	// writes names every create and config update: "create" and "update" for serving
	// compute, "create dedicated" and "update dedicated" for dedicated compute.
	writes []string
	// rawWrites holds the body of every create and config update, as sent.
	rawWrites []string
	// gets counts the GETs of an existing endpoint. Those numbered in failOnGets
	// answer 500, and from readyAfterGets on (when set) the endpoint is ready.
	gets           int
	failOnGets     map[int]bool
	readyAfterGets int
}

func newFakeEndpoints(t *testing.T, existing *servingEndpoint) *fakeEndpoints {
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
		var entities []observedEntity
		for _, e := range req.Config.ServedEntities {
			entities = append(entities, observeServing(e))
		}
		f.endpoint = &servingEndpoint{Name: req.Name, Config: &endpointConfig{ServedEntities: entities}, State: &f.finalState}
	case r.Method == http.MethodPost && r.URL.Path == dedicatedEndpointsPath:
		var req dedicatedEndpointRequest
		if !f.decode(w, r, &req) {
			return
		}
		f.writes = append(f.writes, "create dedicated")
		f.endpoint = &servingEndpoint{Name: req.Name, Config: &endpointConfig{ServedEntities: observeDedicated(req.Config.ServedEntities)}, State: &f.finalState}
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, dedicatedEndpointsPath+"/"):
		var req dedicatedEndpointUpdate
		if !f.decode(w, r, &req) {
			return
		}
		// As the server does, which reads the entities from the request's config.
		if len(req.Config.ServedEntities) == 0 {
			http.Error(w, `{"error_code": "INVALID_PARAMETER_VALUE", "message": "config is required for dedicated inference"}`, http.StatusBadRequest)
			return
		}
		if f.updateConflicts(w) {
			return
		}
		f.writes = append(f.writes, "update dedicated")
		f.endpoint.Config = &endpointConfig{ServedEntities: observeDedicated(req.Config.ServedEntities)}
		f.endpoint.PendingConfig = nil
		f.endpoint.State = &f.finalState
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/config"):
		var req serving.EndpointCoreConfigInput
		if !f.decode(w, r, &req) {
			return
		}
		if f.updateConflicts(w) {
			return
		}
		f.writes = append(f.writes, "update")
		var entities []observedEntity
		for _, e := range req.ServedEntities {
			entities = append(entities, observeServing(e))
		}
		f.endpoint.Config = &endpointConfig{ServedEntities: entities}
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

// updateConflicts answers a config update that arrives while the endpoint is updating.
func (f *fakeEndpoints) updateConflicts(w http.ResponseWriter) bool {
	if !updating(f.endpoint) {
		return false
	}
	w.WriteHeader(http.StatusConflict)
	_, _ = w.Write([]byte(`{"error_code": "RESOURCE_CONFLICT", "message": "The endpoint is being updated."}`))
	return true
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

// observeServing is a served entity with serving compute, as the endpoint reports it.
func observeServing(e serving.ServedEntityInput) observedEntity {
	return observedEntity{
		EntityName:                e.EntityName,
		EntityVersion:             e.EntityVersion,
		WorkloadType:              string(e.WorkloadType),
		MinProvisionedConcurrency: e.MinProvisionedConcurrency,
		MaxProvisionedConcurrency: e.MaxProvisionedConcurrency,
		ScaleToZeroEnabled:        e.ScaleToZeroEnabled,
		EnvironmentVars:           e.EnvironmentVars,
	}
}

// observeDedicated are dedicated served entities, as the endpoint reports them.
func observeDedicated(entities []dedicatedServedEntity) []observedEntity {
	var out []observedEntity
	for _, e := range entities {
		out = append(out, observedEntity{
			EntityName:            e.EntityName,
			EntityVersion:         e.EntityVersion,
			ProvisionedModelUnits: e.ProvisionedModelUnits,
			ComputeProfile:        e.ComputeProfile,
		})
	}
	return out
}

// observe is entity as the endpoint reports it once it serves it.
func observe(entity servedEntity) observedEntity {
	if entity.Dedicated != nil {
		return observedEntity{
			EntityName:            entity.EntityName,
			EntityVersion:         entity.EntityVersion,
			ProvisionedModelUnits: entity.Dedicated.ProvisionedModelUnits,
			ComputeProfile:        entity.Dedicated.ComputeProfile,
		}
	}
	return observedEntity{
		EntityName:                entity.EntityName,
		EntityVersion:             entity.EntityVersion,
		WorkloadType:              entity.Serving.WorkloadType,
		MinProvisionedConcurrency: entity.Serving.MinConcurrency,
		MaxProvisionedConcurrency: entity.Serving.MaxConcurrency,
		ScaleToZeroEnabled:        entity.Serving.ScaleToZero,
	}
}

func (f *fakeEndpoints) start() *httptest.Server {
	srv := httptest.NewServer(f)
	f.t.Cleanup(srv.Close)
	return srv
}

// testEntity is a version of the test model on serving compute: one or two replicas
// of GPU_MEDIUM.
func testEntity(version string) servedEntity {
	return servedEntity{
		EntityName:    "main.models.qwen",
		EntityVersion: version,
		Serving:       servingCompute{WorkloadType: "GPU_MEDIUM", MinConcurrency: 4, MaxConcurrency: 8},
	}
}

// testDedicatedEntity is a version of the test model on dedicated compute.
func testDedicatedEntity(version string) servedEntity {
	return servedEntity{
		EntityName:    "main.models.qwen",
		EntityVersion: version,
		Dedicated:     &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 600},
	}
}

func endpointServing(entity servedEntity, state serving.EndpointState) *servingEndpoint {
	return &servingEndpoint{
		Name:   "qwen-chat",
		Config: &endpointConfig{ServedEntities: []observedEntity{observe(entity)}},
		State:  &state,
	}
}

var readyState = serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}

func TestServeOnEndpointCreatesAMissingEndpoint(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	srv := fake.start()

	ready, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, nil)
	require.NoError(t, err)
	assert.True(t, ready)
	assert.Equal(t, []string{"create"}, fake.writes)
	assert.Equal(t, "3", fake.endpoint.Config.ServedEntities[0].EntityVersion)
}

func TestServeOnEndpointWithoutWatchReturnsOnceAccepted(t *testing.T) {
	tests := []struct {
		name       string
		existing   *servingEndpoint
		wantWrites []string
	}{
		{"a missing endpoint", nil, []string{"create"}},
		{"another version", endpointServing(testEntity("2"), readyState), []string{"update"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEndpoints(t, tt.existing)
			srv := fake.start()
			polls := 0
			ready, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), false, func(*servingEndpoint) { polls++ })
			require.NoError(t, err)
			assert.False(t, ready)
			assert.Equal(t, tt.wantWrites, fake.writes)
			assert.Zero(t, polls, "nothing is waited for")
		})
	}
}

func TestServeOnEndpointWithoutWatchDoesntWaitForItsOwnPendingUpdate(t *testing.T) {
	existing := endpointServing(testEntity("2"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &endpointConfig{ServedEntities: []observedEntity{observe(testEntity("3"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	ready, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), false, nil)
	require.NoError(t, err)
	assert.False(t, ready)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointWithoutWatchReportsAServedVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), readyState))
	srv := fake.start()

	ready, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), false, nil)
	require.NoError(t, err)
	assert.True(t, ready)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointRefusesAnotherModelBeforeWaitingBehindItsUpdate(t *testing.T) {
	other := testEntity("5")
	other.EntityName = "main.models.llama"
	existing := endpointServing(other, serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &endpointConfig{ServedEntities: []observedEntity{observe(other)}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	polls := 0
	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), false, func(*servingEndpoint) { polls++ })
	assert.ErrorContains(t, err, "endpoint qwen-chat serves version 5 of main.models.llama")
	assert.Zero(t, polls)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointUpdatesOnceAnotherUpdateFails(t *testing.T) {
	existing := endpointServing(testEntity("1"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &endpointConfig{ServedEntities: []observedEntity{observe(testEntity("2"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, func(endpoint *servingEndpoint) {
		// The other update fails after the first poll, which frees the endpoint.
		if updating(endpoint) {
			fake.mu.Lock()
			defer fake.mu.Unlock()
			fake.endpoint = endpointServing(testEntity("1"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed})
		}
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
	assert.Equal(t, "3", fake.endpoint.Config.ServedEntities[0].EntityVersion)
}

func TestServeOnEndpointUpdatesAnotherVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("2"), readyState))
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
}

func TestServeOnEndpointUpdatesChangedCompute(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), readyState))
	srv := fake.start()
	entity := testEntity("3")
	entity.Serving.MaxConcurrency = 16

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", entity, true, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"update"}, fake.writes)
}

func TestServeOnEndpointLeavesAServedVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), readyState))
	srv := fake.start()
	// Unity Catalog reports names lowercased.
	entity := testEntity("3")
	entity.EntityName = "Main.Models.Qwen"

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", entity, true, nil)
	require.NoError(t, err)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointCreatesADedicatedEndpoint(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	srv := fake.start()

	ready, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testDedicatedEntity("3"), true, nil)
	require.NoError(t, err)
	assert.True(t, ready)
	assert.Equal(t, []string{"create dedicated"}, fake.writes)
	assert.Equal(t, "COMPUTE_PROFILE_LOW_LATENCY", fake.endpoint.Config.ServedEntities[0].ComputeProfile)
	require.Len(t, fake.rawWrites, 1)
	assert.JSONEq(t, `{
		"name": "qwen-chat",
		"config": {"served_entities": [{
			"entity_name": "main.models.qwen",
			"entity_version": "3",
			"provisioned_model_units": 600,
			"compute_profile": "COMPUTE_PROFILE_LOW_LATENCY"
		}]}
	}`, fake.rawWrites[0])
}

func TestServeOnEndpointUpdatesADedicatedEndpointWhenItsCapacityChanges(t *testing.T) {
	tests := []struct {
		name     string
		entity   servedEntity
		wantBody string
	}{
		{
			"units",
			servedEntity{EntityName: "main.models.qwen", EntityVersion: "3", Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 900}},
			`{"config": {"served_entities": [{"entity_name": "main.models.qwen", "entity_version": "3", "provisioned_model_units": 900, "compute_profile": "COMPUTE_PROFILE_LOW_LATENCY"}]}}`,
		},
		{
			"profile",
			servedEntity{EntityName: "main.models.qwen", EntityVersion: "3", Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_FAST", ProvisionedModelUnits: 600}},
			`{"config": {"served_entities": [{"entity_name": "main.models.qwen", "entity_version": "3", "provisioned_model_units": 600, "compute_profile": "COMPUTE_PROFILE_FAST"}]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEndpoints(t, endpointServing(testDedicatedEntity("3"), readyState))
			srv := fake.start()

			_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", tt.entity, true, nil)
			require.NoError(t, err)
			assert.Equal(t, []string{"update dedicated"}, fake.writes)
			require.Len(t, fake.rawWrites, 1)
			assert.JSONEq(t, tt.wantBody, fake.rawWrites[0])
		})
	}
}

func TestServeOnEndpointLeavesADedicatedVersion(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testDedicatedEntity("3"), readyState))
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testDedicatedEntity("3"), true, nil)
	require.NoError(t, err)
	assert.Empty(t, fake.writes)
}

func TestServeOnEndpointRefusesToChangeAnEndpointsKind(t *testing.T) {
	tests := []struct {
		name     string
		existing servedEntity
		entity   servedEntity
		wantErr  string
	}{
		{"serving compute to dedicated", testEntity("3"), testDedicatedEntity("3"), "endpoint qwen-chat serves on serving compute, and this version is served on dedicated compute"},
		{"dedicated to serving compute", testDedicatedEntity("3"), testEntity("3"), "endpoint qwen-chat serves on dedicated compute, and this version is served on serving compute"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEndpoints(t, endpointServing(tt.existing, readyState))
			srv := fake.start()

			_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", tt.entity, true, nil)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, fake.writes)
		})
	}
}

func TestServeOnEndpointWaitsForItsOwnPendingUpdate(t *testing.T) {
	existing := endpointServing(testEntity("2"), serving.EndpointState{Ready: serving.EndpointStateReadyReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress})
	existing.PendingConfig = &endpointConfig{ServedEntities: []observedEntity{observe(testEntity("3"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	polls := 0
	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, func(*servingEndpoint) {
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
	existing.PendingConfig = &endpointConfig{ServedEntities: []observedEntity{observe(testEntity("2"))}}
	fake := newFakeEndpoints(t, existing)
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, func(endpoint *servingEndpoint) {
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
	split.Config.ServedEntities = append(split.Config.ServedEntities, observe(testEntity("1")))
	tests := []struct {
		name     string
		existing *servingEndpoint
		wantErr  string
	}{
		{"another model", endpointServing(other, readyState), "endpoint qwen-chat serves version 5 of main.models.llama, and air serve replaces only a single version of main.models.qwen"},
		{"a traffic split", split, "endpoint qwen-chat serves version 2 of main.models.qwen and version 1 of main.models.qwen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEndpoints(t, tt.existing)
			w := newTestWorkspaceClient(t, fake.start().URL)

			_, err := serveOnEndpoint(withDeadline(t), w, "qwen-chat", testEntity("3"), true, nil)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, fake.writes)
		})
	}
}

func TestServeOnEndpointWaitsThroughTransientFailures(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	// The endpoint starts over three polls, and the first GET of each fails: more
	// failures than one request may retry, spread over the wait.
	fake.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateInProgress}
	fake.failOnGets = map[int]bool{1: true, 3: true, 5: true}
	fake.readyAfterGets = 6
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"create"}, fake.writes)
	assert.Equal(t, 6, fake.gets)
}

func TestServeOnEndpointReportsAFailedUpdate(t *testing.T) {
	fake := newFakeEndpoints(t, nil)
	fake.finalState = serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed}
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, nil)
	assert.ErrorContains(t, err, "UPDATE_FAILED")
}

func TestServeOnEndpointReportsAnEndpointThatIsNotReady(t *testing.T) {
	fake := newFakeEndpoints(t, endpointServing(testEntity("3"), serving.EndpointState{Ready: serving.EndpointStateReadyNotReady, ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating}))
	srv := fake.start()

	_, err := serveOnEndpoint(withDeadline(t), newTestWorkspaceClient(t, srv.URL), "qwen-chat", testEntity("3"), true, nil)
	assert.ErrorContains(t, err, "endpoint qwen-chat serves version 3 of main.models.qwen but isn't ready")
}

func TestDedicatedEntityJSON(t *testing.T) {
	entity := testDedicatedEntity("3")
	raw, err := json.Marshal(entity.dedicatedJSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"entity_name": "main.models.qwen", "entity_version": "3", "provisioned_model_units": 600, "compute_profile": "COMPUTE_PROFILE_LOW_LATENCY"}`, string(raw))
}

func TestServesOnly(t *testing.T) {
	entity := testEntity("3")
	assert.True(t, servesOnly([]observedEntity{observe(entity)}, entity))
	assert.False(t, servesOnly(nil, entity))
	assert.False(t, servesOnly([]observedEntity{observe(entity), observe(testEntity("2"))}, entity))

	// air serve sets no environment variables, so an endpoint with any is updated.
	withVars := observe(entity)
	withVars.EnvironmentVars = map[string]string{"HF_TOKEN": "{{secrets/llm/hf_token}}"}
	assert.False(t, servesOnly([]observedEntity{withVars}, entity))

	// Where GPU APIs are off, the endpoint leaves out a CPU entity's workload type.
	cpu := testEntity("3")
	cpu.Serving.WorkloadType = "CPU"
	reported := observe(cpu)
	reported.WorkloadType = ""
	assert.True(t, servesOnly([]observedEntity{reported}, cpu))
	assert.False(t, servesOnly([]observedEntity{reported}, entity), "a missing workload type is CPU, not a GPU")

	scaled := observe(entity)
	scaled.ScaleToZeroEnabled = true
	assert.False(t, servesOnly([]observedEntity{scaled}, entity))

	for _, observed := range []func(*observedEntity){
		func(o *observedEntity) { o.MinProvisionedConcurrency = 8 },
		func(o *observedEntity) { o.MaxProvisionedConcurrency = 16 },
		func(o *observedEntity) { o.WorkloadType = "GPU_XLARGE" },
	} {
		other := observe(entity)
		observed(&other)
		assert.False(t, servesOnly([]observedEntity{other}, entity), "%+v", other)
	}
}

func TestServesOnlyDedicated(t *testing.T) {
	entity := testDedicatedEntity("3")
	assert.True(t, servesOnly([]observedEntity{observe(entity)}, entity))
	assert.False(t, servesOnly([]observedEntity{observe(testEntity("3"))}, entity), "serving compute isn't a dedicated entity")
	assert.False(t, servesOnly([]observedEntity{observe(entity)}, testEntity("3")), "a dedicated entity isn't serving compute")
}
