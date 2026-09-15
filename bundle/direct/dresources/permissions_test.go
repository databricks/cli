package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The object_id reference is synthesized from the resource type, so it is only exercised
// end-to-end by acceptance tests. Pin both the default id field and an override here.
func TestPermissionsPrepareInputConfig(t *testing.T) {
	tests := []struct {
		resourceType string
		resourceKey  string
		expectedRef  string
	}{
		{
			resourceType: "jobs.permissions",
			resourceKey:  "resources.jobs.foo.permissions",
			expectedRef:  "/jobs/${resources.jobs.foo.id}",
		},
		{
			// models use a numeric model_id, not the model name recorded as the CRUD state ID
			resourceType: "models.permissions",
			resourceKey:  "resources.models.foo.permissions",
			expectedRef:  "/registered-models/${resources.models.foo.model_id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			adapter, err := NewAdapter(SupportedResources[tt.resourceType], tt.resourceType, nil)
			require.NoError(t, err)

			input := &[]resources.JobPermission{{Level: "CAN_VIEW", UserName: "alice"}}
			sv, err := adapter.PrepareInputConfig(input, tt.resourceKey)
			require.NoError(t, err)

			assert.Equal(t, &PermissionsState{
				EmbeddedSlice: []StatePermission{{Level: "CAN_VIEW", UserName: "alice"}},
			}, sv.Value)
			assert.Equal(t, map[string]string{"object_id": tt.expectedRef}, sv.Refs)
		})
	}
}

// Configure resolves the parent type at init, so an unregistered parent fails before planning.
func TestPermissionsConfigureUnsupportedParent(t *testing.T) {
	_, err := NewAdapter(SupportedResources["jobs.permissions"], "unregistered.permissions", nil)
	assert.ErrorContains(t, err, "unsupported permissions resource type: unregistered")
}

// Resources without PrepareInputConfig get their config through unchanged and add no references.
func TestPrepareInputConfigPassthrough(t *testing.T) {
	adapter, err := NewAdapter(SupportedResources["schemas"], "schemas", nil)
	require.NoError(t, err)

	input := &resources.Schema{CreateSchema: catalog.CreateSchema{Name: "myschema"}}
	sv, err := adapter.PrepareInputConfig(input, "resources.schemas.foo")
	require.NoError(t, err)

	assert.Same(t, input, sv.Value)
	assert.Nil(t, sv.Refs)
}
