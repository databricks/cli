package direct_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareBindingsImportsConfiguredRemoteFields(t *testing.T) {
	for _, test := range []struct {
		name     string
		resource string
		path     string
		config   string
		remote   string
		state    string
	}{
		{
			name:     "immutable backend defaults",
			resource: "experiments",
			path:     "/api/2.0/mlflow/experiments/get",
			config:   `{"name":"desired"}`,
			remote:   `{"experiment":{"experiment_id":"123","name":"existing","artifact_location":"dbfs:/existing"}}`,
			state:    `{"name":"existing"}`,
		},
		{
			name:     "remote etag for drift detection",
			resource: "genie_spaces",
			path:     "/api/2.0/genie/spaces/123",
			config:   `{"title":"desired"}`,
			remote:   `{"space_id":"123","title":"existing","etag":"remote-version","parent_path":"/Shared"}`,
			state:    `{"title":"existing","etag":"remote-version"}`,
		},
		{
			name:     "keyed arrays, maps, explicit zero, and large integers",
			resource: "jobs",
			path:     "/api/2.2/jobs/get",
			config:   `{"name":"desired","timeout_seconds":0,"tasks":[{"task_key":"a","notebook_task":{"notebook_path":"/a","base_parameters":{"keep":"new"}}},{"task_key":"b","description":"new"}]}`,
			remote:   `{"job_id":123,"settings":{"name":"existing","timeout_seconds":9007199254740993,"tasks":[{"task_key":"b","description":"old","timeout_seconds":50},{"task_key":"a","notebook_task":{"notebook_path":"/a","base_parameters":{"keep":"old","drop":"value"}}},{"task_key":"unused"}]}}`,
			state:    `{"name":"existing","timeout_seconds":9007199254740993,"tasks":[{"task_key":"b","description":"old"},{"task_key":"a","notebook_task":{"notebook_path":"/a","base_parameters":{"keep":"old"}}}]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", test.path, func(req testserver.Request) any {
				return json.RawMessage(test.remote)
			})
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)
			var resource map[string]any
			require.NoError(t, json.Unmarshal([]byte(test.config), &resource))
			resource["lifecycle"] = map[string]any{"bind": map[string]string{"id": "123"}}
			source, err := json.Marshal(map[string]any{"resources": map[string]any{test.resource: map[string]any{"test": resource}}})
			require.NoError(t, err)
			root, diagnostics := config.LoadFromBytes("databricks.yml", source)
			require.Empty(t, diagnostics)
			var deployment direct.DeploymentBundle
			deployment.StateDB.OpenWithData(filepath.Join(t.TempDir(), "resources.json"), dstate.NewDatabase("", 0))
			bindings, err := deployment.PrepareBindings(t.Context(), client, root, nil)
			require.NoError(t, err)
			assert.JSONEq(t, test.state, string(bindings["resources."+test.resource+".test"].State))
			if test.resource == "jobs" {
				assert.Contains(t, string(bindings["resources.jobs.test"].State), "9007199254740993")
			}
			assert.Empty(t, deployment.StateDB.Data.State)
		})
	}
}
