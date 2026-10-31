package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/auth"
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

func TestPipelineRawRequests(t *testing.T) {
	server := testserver.New(t)
	var bodies, workspaceIDs []string
	server.Handle("POST", "/api/2.0/pipelines", func(req testserver.Request) any {
		bodies = append(bodies, string(req.Body))
		workspaceIDs = append(workspaceIDs, req.Headers.Get(auth.WorkspaceIDHeader))
		return pipelines.CreatePipelineResponse{PipelineId: "p1"}
	})
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		workspaceIDs = append(workspaceIDs, req.Headers.Get(auth.WorkspaceIDHeader))
		return map[string]any{"pipeline_id": "p1", "spec": map[string]any{"name": "orders"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token", WorkspaceID: "123"})
	require.NoError(t, err)
	r := (&ResourcePipeline{}).New(w)

	testing := &resources.PipelineTesting{Sources: []resources.PipelineTestSource{{Glob: &pipelines.PathPattern{Include: "/Workspace/orders/tests/**"}}}}
	_, _, err = r.DoCreate(t.Context(), &PipelineState{Name: "orders", Testing: testing})
	require.NoError(t, err)
	_, err = r.DoRead(t.Context(), "p1")
	require.NoError(t, err)
	// Without testing, the request is the SDK's.
	_, _, err = r.DoCreate(t.Context(), &PipelineState{Name: "orders"})
	require.NoError(t, err)

	// The raw requests carry the workspace ID, as the SDK's do.
	assert.Equal(t, []string{"123", "123", "123"}, workspaceIDs)
	assert.JSONEq(t, `{"name": "orders"}`, bodies[1])
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
