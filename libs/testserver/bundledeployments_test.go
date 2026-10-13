package testserver_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateVersionResourceTypes checks the resource_key validation shared with DMS.
func TestCreateVersionResourceTypes(t *testing.T) {
	tests := []struct {
		resourceKey string
		supported   bool
	}{
		{resourceKey: "jobs.my_job", supported: true},
		{resourceKey: "pipelines.my_pipeline", supported: true},
		{resourceKey: "postgres_roles.my_role", supported: true},
		{resourceKey: "secrets.my_secret", supported: true},
		{resourceKey: "cluster_policies.my_policy", supported: true},
		{resourceKey: "jobs.name.with.dots", supported: true},
		{resourceKey: ""},
		{resourceKey: "jobs"},
		{resourceKey: ".my_resource"},
		{resourceKey: "unknown.my_resource"},
		{resourceKey: "mcp_services.my_service"},
		{resourceKey: "model_provider_services.my_service"},
		{resourceKey: "model_services.my_service"},
		{resourceKey: "postgres_snapshot_schedules.my_schedule"},
		{resourceKey: "resources.jobs.my_job"},
		{resourceKey: "JOBS.my_job"},
	}
	for _, tt := range tests {
		t.Run(tt.resourceKey, func(t *testing.T) {
			deployment := &testserver.DmsDeployment{
				Versions:   map[string]*bundledeployments.Version{},
				Resources:  map[string]bundledeployments.Resource{},
				Operations: map[string]*bundledeployments.Operation{},
			}
			ws := &testserver.FakeWorkspace{
				DmsDeployments: map[string]*testserver.DmsDeployment{"123": deployment},
			}
			body, err := json.Marshal(map[string]any{
				"operations": []map[string]string{
					{"resource_key": "jobs.other", "action_type": "OPERATION_ACTION_TYPE_CREATE"},
					{"resource_key": tt.resourceKey, "action_type": "OPERATION_ACTION_TYPE_CREATE"},
				},
			})
			require.NoError(t, err)
			resp := ws.CreateVersion(testserver.Request{
				URL:  &url.URL{RawQuery: "version_id=1"},
				Body: body,
			}, "123")

			if !tt.supported {
				require.Equal(t, 400, resp.StatusCode)
				message := "operations.resource_key must have a known resource type prefix (e.g. 'jobs.', 'pipelines.'): " + tt.resourceKey
				if tt.resourceKey == "" {
					message = "operations.resource_key is required"
				}
				assert.Equal(t, map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    message,
				}, resp.Body)
				assert.Empty(t, deployment.Deployment.LastVersionId)
				assert.Empty(t, deployment.Versions)
				assert.Empty(t, deployment.Operations)
				assert.Empty(t, deployment.Resources)
				return
			}

			require.Zero(t, resp.StatusCode)
			assert.Equal(t, "1", deployment.Deployment.LastVersionId)
			assert.Contains(t, deployment.Versions, "1")
			assert.Len(t, deployment.Operations, 2)
			assert.Contains(t, deployment.Operations, "deployments/123/versions/1/operations/"+tt.resourceKey)
		})
	}
}
