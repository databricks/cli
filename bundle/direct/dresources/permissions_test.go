package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/testserver"
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
			adapter, err := NewAdapter(SupportedResources()[tt.resourceType], tt.resourceType, nil)
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
// Secret scopes declare permissions but not the permissions API, so they are unsupported too.
func TestPermissionsConfigureUnsupportedParent(t *testing.T) {
	for _, parent := range []string{"unregistered", "schemas", "secret_scopes"} {
		_, err := NewAdapter(SupportedResources()["jobs.permissions"], parent+".permissions", nil)
		assert.ErrorContains(t, err, "unsupported permissions resource type: "+parent)
	}
}

// Types that declare a permissions API object type get the shared adapter; the others must
// register their own.
func TestPermissionsAdaptersMatchDeclarations(t *testing.T) {
	declared := map[string]bool{}
	for key, d := range config.ResourceTypesImplementing[resources.PermissionsDeclarer]() {
		resourceType := key + ".permissions"
		declared[resourceType] = true
		adapter, ok := SupportedResources()[resourceType]
		if !assert.True(t, ok, "%s has no adapter", resourceType) {
			continue
		}
		usesAPI := d.PermissionsSupport().RequestObjectType != ""
		_, isPermissions := adapter.(*ResourcePermissions)
		assert.Equal(t, usesAPI, isPermissions, "%s: the permissions API adapter must be used exactly when the type declares RequestObjectType", resourceType)
	}
	for resourceType := range SupportedResources() {
		if strings.HasSuffix(resourceType, ".permissions") {
			assert.True(t, declared[resourceType], "%s has an adapter but its parent does not declare permissions", resourceType)
		}
	}
}

// The object ID reference reads ObjectIDField from the parent's remote state.
// The deployment framework supplies "id" itself.
func TestPermissionsObjectIDFieldIsReadable(t *testing.T) {
	for key, d := range config.ResourceTypesImplementing[resources.PermissionsDeclarer]() {
		field := d.PermissionsSupport().ObjectIDField
		if field == "" || field == "id" {
			continue
		}
		adapter, err := NewAdapter(SupportedResources()[key], key, nil)
		require.NoError(t, err)
		assert.NoError(t, structaccess.ValidateByString(adapter.RemoteType(), field), key)
	}
}

// The fake server declares its own permissions object types, since it does not depend
// on the bundle configuration. Each one is registered with registerPermissionsObjectType
// from the file in libs/testserver that serves its API.
func TestPermissionsObjectTypesMatchTestServer(t *testing.T) {
	for key, d := range config.ResourceTypesImplementing[resources.PermissionsDeclarer]() {
		s := d.PermissionsSupport()
		if s.RequestObjectType == "" {
			continue
		}
		objectType, ok := testserver.PermissionsObjectType(s.RequestObjectType)
		assert.True(t, ok, "%s: the test server does not serve permissions for %q; register it with registerPermissionsObjectType in the libs/testserver file of its API", key, s.RequestObjectType)
		assert.Equal(t, s.ResponseObjectType, objectType, key)
	}
}

// Resources without PrepareInputConfig get their config through unchanged and add no references.
func TestPrepareInputConfigPassthrough(t *testing.T) {
	adapter, err := NewAdapter(SupportedResources()["schemas"], "schemas", nil)
	require.NoError(t, err)

	input := &resources.Schema{Name: "myschema"}
	sv, err := adapter.PrepareInputConfig(input, "resources.schemas.foo")
	require.NoError(t, err)

	assert.Same(t, input, sv.Value)
	assert.Nil(t, sv.Refs)
}
