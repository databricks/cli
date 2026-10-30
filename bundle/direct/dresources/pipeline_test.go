package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipelineRemote verifies that all fields from pipelines.GetPipelineResponse
// (except Spec and internal fields) are present in PipelineRemote.
func TestPipelineRemote(t *testing.T) {
	assertFieldsCovered(t, reflect.TypeFor[pipelines.GetPipelineResponse](), reflect.TypeFor[PipelineRemote](), map[string]bool{
		"Spec":            true, // Embedded as pipelines.CreatePipeline (via makePipelineRemote)
		"ForceSendFields": true, // Internal marshaling field
		"Name":            true, // Available through embedded CreatePipeline
		"Parameters":      true, // Available through embedded CreatePipeline
		"RunAs":           true, // Available through embedded CreatePipeline
	})
}

func TestPipelineOverrideChangeDescCascadeOnDestroy(t *testing.T) {
	r := &ResourcePipeline{}
	cascadePath := structpath.MustParsePath("cascade_on_destroy")
	ptr := func(b bool) *bool { return &b }

	t.Run("unset forces state-only update", func(t *testing.T) {
		change := &ChangeDesc{Action: deployplan.Skip, Old: ptr(false), New: nil}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), cascadePath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})

	t.Run("changed value forces state-only update", func(t *testing.T) {
		change := &ChangeDesc{Action: deployplan.Skip, Old: ptr(true), New: ptr(false)}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), cascadePath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})

	t.Run("unchanged value is left as skip", func(t *testing.T) {
		change := &ChangeDesc{Action: deployplan.Skip, Old: ptr(false), New: ptr(false)}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), cascadePath, change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
	})

	t.Run("other paths are untouched", func(t *testing.T) {
		namePath := structpath.MustParsePath("name")
		change := &ChangeDesc{Action: deployplan.Skip, Old: "a", New: nil}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), namePath, change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
	})
}

func TestPipelineCreateAndUpdateSendTesting(t *testing.T) {
	server := testserver.New(t)
	var bodies []string
	server.Handle("POST", "/api/2.0/pipelines", func(req testserver.Request) any {
		bodies = append(bodies, string(req.Body))
		return pipelines.CreatePipelineResponse{PipelineId: "p1"}
	})
	server.Handle("PUT", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		bodies = append(bodies, string(req.Body))
		return map[string]any{}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	r := (&ResourcePipeline{}).New(w)

	testConfig := &resources.PipelineTesting{
		Sources:     []resources.PipelineTestSource{{Glob: &pipelines.PathPattern{Include: "/Workspace/orders/tests/**"}}},
		Environment: &resources.PipelineTestEnvironment{Dependencies: []string{"--editable /Workspace/orders[test]"}},
	}
	state := &PipelineState{Name: "orders", Testing: testConfig}
	id, _, err := r.DoCreate(t.Context(), state)
	require.NoError(t, err)
	assert.Equal(t, "p1", id)
	_, err = r.DoUpdate(t.Context(), id, state, &PlanEntry{Changes: Changes{"testing": &ChangeDesc{Action: deployplan.Update}}})
	require.NoError(t, err)

	// Without testing, the requests are the SDK's.
	state.Testing = nil
	_, _, err = r.DoCreate(t.Context(), state)
	require.NoError(t, err)
	_, err = r.DoUpdate(t.Context(), id, state, &PlanEntry{Changes: Changes{"testing": &ChangeDesc{Action: deployplan.Update}}})
	require.NoError(t, err)

	wantTesting := `"testing": {
		"sources": [{"glob": {"include": "/Workspace/orders/tests/**"}}],
		"environment": {"dependencies": ["--editable /Workspace/orders[test]"]}
	}`
	require.Len(t, bodies, 4)
	assert.JSONEq(t, `{"name": "orders", `+wantTesting+`}`, bodies[0])
	assert.JSONEq(t, `{"name": "orders", `+wantTesting+`}`, bodies[1])
	assert.JSONEq(t, `{"name": "orders"}`, bodies[2])
	assert.JSONEq(t, `{"name": "orders"}`, bodies[3])
}

func TestPipelineReadReturnsTesting(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		return map[string]any{
			"pipeline_id": req.Vars["pipeline_id"],
			"state":       "IDLE",
			"spec": map[string]any{
				"name":    "orders",
				"testing": map[string]any{"sources": []any{map[string]any{"glob": map[string]any{"include": "/Workspace/orders/tests/**"}}}},
			},
		}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	r := (&ResourcePipeline{}).New(w)

	remote, err := r.DoRead(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "p1", remote.PipelineId)
	assert.Equal(t, "orders", remote.Name)
	assert.Equal(t, pipelines.PipelineStateIdle, remote.State)
	require.NotNil(t, remote.Testing)
	require.Len(t, remote.Testing.Sources, 1)
	assert.Equal(t, "/Workspace/orders/tests/**", remote.Testing.Sources[0].Glob.Include)
	assert.Same(t, remote.Testing, r.RemapState(remote).Testing)
}
