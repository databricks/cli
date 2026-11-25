package dresources

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	registerWithoutTestFixture("pipelines", "the test server creates a pipeline from an empty request")

	registerTestDeps("pipelines.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.Pipelines.Create(ctx, pipelines.CreatePipeline{
			Name: "pipeline-permissions",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/pipelines/" + resp.PipelineId,
			EmbeddedSlice: []StatePermission{{
				// Pipelines require exactly one owner, like jobs.
				Level:    "IS_OWNER",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInRemoteType("pipelines",
		// Note that this is a deliberate omission: cascade_on_destroy is a client-side-only field.
		// It has no remote counterpart, so it is never expected in RemoteType.
		"cascade_on_destroy",
	)

	registerCreateRecoveryOptOut("pipelines", "the backend assigns the pipeline id, names are not unique, and the create request takes no idempotency token")
}

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
