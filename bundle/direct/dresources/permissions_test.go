package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structaccess"
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
			resourceType: "alerts.permissions",
			resourceKey:  "resources.alerts.foo.permissions",
			expectedRef:  "/alertsv2/${resources.alerts.foo.id}",
		},
		{
			resourceType: "apps.permissions",
			resourceKey:  "resources.apps.foo.permissions",
			expectedRef:  "/apps/${resources.apps.foo.id}",
		},
		{
			resourceType: "clusters.permissions",
			resourceKey:  "resources.clusters.foo.permissions",
			expectedRef:  "/clusters/${resources.clusters.foo.id}",
		},
		{
			resourceType: "cluster_policies.permissions",
			resourceKey:  "resources.cluster_policies.foo.permissions",
			expectedRef:  "/cluster-policies/${resources.cluster_policies.foo.id}",
		},
		{
			resourceType: "instance_pools.permissions",
			resourceKey:  "resources.instance_pools.foo.permissions",
			expectedRef:  "/instance-pools/${resources.instance_pools.foo.id}",
		},
		{
			resourceType: "dashboards.permissions",
			resourceKey:  "resources.dashboards.foo.permissions",
			expectedRef:  "/dashboards/${resources.dashboards.foo.id}",
		},
		{
			resourceType: "genie_spaces.permissions",
			resourceKey:  "resources.genie_spaces.foo.permissions",
			expectedRef:  "/genie/${resources.genie_spaces.foo.id}",
		},
		{
			resourceType: "database_instances.permissions",
			resourceKey:  "resources.database_instances.foo.permissions",
			expectedRef:  "/database-instances/${resources.database_instances.foo.id}",
		},
		{
			resourceType: "postgres_projects.permissions",
			resourceKey:  "resources.postgres_projects.foo.permissions",
			expectedRef:  "/database-projects/${resources.postgres_projects.foo.project_id}",
		},
		{
			resourceType: "jobs.permissions",
			resourceKey:  "resources.jobs.foo.permissions",
			expectedRef:  "/jobs/${resources.jobs.foo.id}",
		},
		{
			resourceType: "experiments.permissions",
			resourceKey:  "resources.experiments.foo.permissions",
			expectedRef:  "/experiments/${resources.experiments.foo.id}",
		},
		{
			resourceType: "models.permissions",
			resourceKey:  "resources.models.foo.permissions",
			expectedRef:  "/registered-models/${resources.models.foo.model_id}",
		},
		{
			resourceType: "model_serving_endpoints.permissions",
			resourceKey:  "resources.model_serving_endpoints.foo.permissions",
			expectedRef:  "/serving-endpoints/${resources.model_serving_endpoints.foo.endpoint_id}",
		},
		{
			resourceType: "pipelines.permissions",
			resourceKey:  "resources.pipelines.foo.permissions",
			expectedRef:  "/pipelines/${resources.pipelines.foo.id}",
		},
		{
			resourceType: "sql_warehouses.permissions",
			resourceKey:  "resources.sql_warehouses.foo.permissions",
			expectedRef:  "/sql/warehouses/${resources.sql_warehouses.foo.id}",
		},
		{
			resourceType: "vector_search_endpoints.permissions",
			resourceKey:  "resources.vector_search_endpoints.foo.permissions",
			expectedRef:  "/vector-search-endpoints/${resources.vector_search_endpoints.foo.endpoint_uuid}",
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
	for _, parent := range []string{"unregistered", "features", "secret_scopes"} {
		t.Run(parent, func(t *testing.T) {
			_, err := NewAdapter(SupportedResources["jobs.permissions"], parent+".permissions", nil)
			assert.ErrorContains(t, err, "unsupported permissions resource type: "+parent)
		})
	}
}

// Resources without PrepareInputConfig get their config through unchanged and add no references.
func TestPrepareInputConfigPassthrough(t *testing.T) {
	adapter, err := NewAdapter(SupportedResources["schemas"], "schemas", nil)
	require.NoError(t, err)

	input := &resources.Schema{Name: "myschema"}
	sv, err := adapter.PrepareInputConfig(input, "resources.schemas.foo")
	require.NoError(t, err)

	assert.Same(t, input, sv.Value)
	assert.Nil(t, sv.Refs)
}

func TestPermissionsAdapterCoverage(t *testing.T) {
	expected := map[string]any{
		"alerts.permissions":                  (*ResourcePermissions)(nil),
		"apps.permissions":                    (*ResourcePermissions)(nil),
		"clusters.permissions":                (*ResourcePermissions)(nil),
		"cluster_policies.permissions":        (*ResourcePermissions)(nil),
		"instance_pools.permissions":          (*ResourcePermissions)(nil),
		"dashboards.permissions":              (*ResourcePermissions)(nil),
		"genie_spaces.permissions":            (*ResourcePermissions)(nil),
		"database_instances.permissions":      (*ResourcePermissions)(nil),
		"postgres_projects.permissions":       (*ResourcePermissions)(nil),
		"jobs.permissions":                    (*ResourcePermissions)(nil),
		"experiments.permissions":             (*ResourcePermissions)(nil),
		"models.permissions":                  (*ResourcePermissions)(nil),
		"model_serving_endpoints.permissions": (*ResourcePermissions)(nil),
		"pipelines.permissions":               (*ResourcePermissions)(nil),
		"sql_warehouses.permissions":          (*ResourcePermissions)(nil),
		"vector_search_endpoints.permissions": (*ResourcePermissions)(nil),
		"secret_scopes.permissions":           (*ResourceSecretScopeAcls)(nil),
	}
	actual := make(map[string]any)
	for name, resource := range SupportedResources {
		if strings.HasSuffix(name, ".permissions") {
			actual[name] = resource
			assert.Contains(t, config.ResourcesTypes, name)
		}
	}
	assert.Equal(t, expected, actual)
	for name := range config.ResourcesTypes {
		if strings.HasSuffix(name, ".permissions") {
			assert.Contains(t, actual, name)
		}
	}
}

func TestPermissionsObjectIDsAreReferenceable(t *testing.T) {
	for name, description := range resources.All {
		if description.Permissions.RequestObjectType == "" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			resource, ok := SupportedResources[name]
			require.True(t, ok)
			adapter, err := NewAdapter(resource, name, nil)
			require.NoError(t, err)
			// The deployment framework supplies .id; structaccess excludes this read-only field.
			if description.Permissions.ObjectIDField == "id" {
				field, ok := config.ResourcesTypes[name].FieldByName("ID")
				require.True(t, ok)
				jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				assert.Equal(t, "id", jsonName)
				return
			}
			assert.NoError(t, structaccess.ValidateByString(adapter.RemoteType(), description.Permissions.ObjectIDField))
		})
	}
}
