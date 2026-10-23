package migrate_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootFromYAML builds a config.Root from a YAML snippet.
// Template strings like "${resources.jobs.src.name}" are preserved in the
// internal dyn.Value so BuildStateFromTF can find them via ExtractReferences.
func rootFromYAML(t *testing.T, yaml string) config.Root {
	t.Helper()
	v, err := yamlloader.LoadYAML("test", bytes.NewBufferString(yaml))
	require.NoError(t, err)
	var root config.Root
	require.NoError(t, convert.ToTyped(&root, v))
	require.NoError(t, root.Mutate(func(_ dyn.Value) (dyn.Value, error) { return v, nil }))
	return root
}

func runBuildStateFromTF(
	t *testing.T,
	yaml string,
	tfAttrs migrate.TFStateAttrs,
	tfIDs map[string]string,
) map[string]dstate.ResourceEntry {
	t.Helper()

	root := rootFromYAML(t, yaml)
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	statePath := filepath.Join(t.TempDir(), "resources.json")

	var db dstate.DeploymentState
	db.OpenWithData(statePath, dstate.NewDatabase("lineage", 1))
	require.NoError(t, db.UpgradeToWrite())

	_, err = migrate.BuildStateFromTF(t.Context(), &root, adapters, &db, tfAttrs, tfIDs, "")
	require.NoError(t, err)

	_, err = db.Finalize(t.Context())
	require.NoError(t, err)

	raw, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var data dstate.Database
	require.NoError(t, json.Unmarshal(raw, &data))
	return data.State
}

func TestBuildStateFromTF(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		tfAttrs      migrate.TFStateAttrs
		tfIDs        map[string]string
		wantKey      string         // primary key to assert on
		absentKey    string         // key that must NOT be in state
		wantID       string         // expected entry.ID
		wantState    map[string]any // expected fields in the state JSON (parsed via json.Unmarshal)
		wantStateRaw string         // expected substring in raw state JSON bytes (use for large integers)
		wantDeps     []deployplan.DependsOnEntry
	}{
		{
			name: "basic job stored with ID",
			yaml: `
resources:
  jobs:
    my_job:
      name: "hello"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_job": {"my_job": json.RawMessage(`{"id": "123", "name": "hello"}`)},
			},
			tfIDs:   map[string]string{"resources.jobs.my_job": "123"},
			wantKey: "resources.jobs.my_job",
			wantID:  "123",
		},
		{
			name: "resource not in TF state is skipped",
			yaml: `
resources:
  jobs:
    new_job:
      name: "new"
    existing_job:
      name: "existing"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_job": {"existing_job": json.RawMessage(`{"id": "456", "name": "existing"}`)},
			},
			tfIDs:     map[string]string{"resources.jobs.existing_job": "456"},
			wantKey:   "resources.jobs.existing_job",
			absentKey: "resources.jobs.new_job",
			wantID:    "456",
		},
		{
			name: "cross-resource ref: depends_on computed, field resolved",
			yaml: `
resources:
  pipelines:
    src:
      name: "source-pipeline"
  jobs:
    dst:
      name: "${resources.pipelines.src.name}"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_pipeline": {"src": json.RawMessage(`{"id": "p1", "name": "source-pipeline"}`)},
				"databricks_job":      {"dst": json.RawMessage(`{"id": "j1", "name": "source-pipeline"}`)},
			},
			tfIDs: map[string]string{
				"resources.pipelines.src": "p1",
				"resources.jobs.dst":      "j1",
			},
			wantKey:   "resources.jobs.dst",
			wantID:    "j1",
			wantState: map[string]any{"name": "source-pipeline"},
			wantDeps: []deployplan.DependsOnEntry{
				{Node: "resources.pipelines.src", Label: "${resources.pipelines.src.name}"},
			},
		},
		{
			name: "numeric field reference stored as number not string",
			yaml: `
resources:
  jobs:
    src_job:
      name: "source"
      max_concurrent_runs: 4
    dst_job:
      name: "dest"
      max_concurrent_runs: "${resources.jobs.src_job.max_concurrent_runs}"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_job": {
					"src_job": json.RawMessage(`{"id": "111", "name": "source", "max_concurrent_runs": 4}`),
					"dst_job": json.RawMessage(`{"id": "222", "name": "dest",   "max_concurrent_runs": 4}`),
				},
			},
			tfIDs: map[string]string{
				"resources.jobs.src_job": "111",
				"resources.jobs.dst_job": "222",
			},
			wantKey:   "resources.jobs.dst_job",
			wantID:    "222",
			wantState: map[string]any{"max_concurrent_runs": float64(4)},
			wantDeps:  []deployplan.DependsOnEntry{{Node: "resources.jobs.src_job"}},
		},
		{
			// 2^53+1 = 9007199254740993 cannot be represented exactly as float64.
			// json.Unmarshal into map[string]any would produce 9007199254740992 (off by one).
			// UseNumber preserves the original decimal string, so the value is exact.
			name: "large integer run_job_task.job_id preserved exactly",
			yaml: `
resources:
  jobs:
    trigger_job:
      name: "trigger"
    watcher_job:
      name: "watcher"
      tasks:
        - task_key: run_trigger
          run_job_task:
            job_id: "${resources.jobs.trigger_job.id}"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_job": {
					"trigger_job": json.RawMessage(`{"id": "9007199254740993", "name": "trigger", "max_concurrent_runs": 1}`),
					"watcher_job": json.RawMessage(`{"id": "100", "name": "watcher", "task": [{"task_key": "run_trigger", "run_job_task": [{"job_id": 9007199254740993}]}]}`),
				},
			},
			tfIDs: map[string]string{
				"resources.jobs.trigger_job": "9007199254740993",
				"resources.jobs.watcher_job": "100",
			},
			wantKey: "resources.jobs.watcher_job",
			wantID:  "100",
			// job_id must be stored as 9007199254740993, not 9007199254740992 (float64 truncation).
			// Check raw bytes because json.Unmarshal would silently re-truncate when reading back.
			wantStateRaw: `"job_id": 9007199254740993`,
		},
		{
			// model_serving_endpoints permissions reference object_id via the parent's
			// "endpoint_id", which is absent from the parent's TF state attributes. The
			// object_id must come from the permissions node's own TF state ID instead.
			name: "model_serving_endpoints permissions object_id from node ID",
			yaml: `
resources:
  model_serving_endpoints:
    foo:
      name: my-endpoint
      permissions:
        - level: CAN_VIEW
          group_name: users
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_model_serving": {"foo": json.RawMessage(`{"id": "my-endpoint", "name": "my-endpoint"}`)},
			},
			tfIDs: map[string]string{
				"resources.model_serving_endpoints.foo":             "my-endpoint",
				"resources.model_serving_endpoints.foo.permissions": "/serving-endpoints/abc123",
			},
			wantKey:   "resources.model_serving_endpoints.foo.permissions",
			wantID:    "/serving-endpoints/abc123",
			wantState: map[string]any{"object_id": "/serving-endpoints/abc123"},
			wantDeps:  []deployplan.DependsOnEntry{{Node: "resources.model_serving_endpoints.foo"}},
		},
		{
			// database_instances permissions reference object_id via the parent's "id",
			// which is absent from the parent's TF state attributes (it is stored under
			// "name"). The object_id must come from the permissions node's own TF state ID.
			name: "database_instances permissions object_id from node ID",
			yaml: `
resources:
  database_instances:
    foo:
      name: my-db-instance
      permissions:
        - level: CAN_USE
          group_name: users
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_database_instance": {"foo": json.RawMessage(`{"name": "my-db-instance"}`)},
			},
			tfIDs: map[string]string{
				"resources.database_instances.foo":             "my-db-instance",
				"resources.database_instances.foo.permissions": "/database-instances/my-db-instance",
			},
			wantKey:   "resources.database_instances.foo.permissions",
			wantID:    "/database-instances/my-db-instance",
			wantState: map[string]any{"object_id": "/database-instances/my-db-instance"},
			wantDeps:  []deployplan.DependsOnEntry{{Node: "resources.database_instances.foo"}},
		},
		{
			name: "dashboard etag stored from TF attributes",
			yaml: `
resources:
  dashboards:
    my_dash:
      display_name: "My Dashboard"
`,
			tfAttrs: migrate.TFStateAttrs{
				"databricks_dashboard": {
					"my_dash": json.RawMessage(`{"id": "d1", "display_name": "My Dashboard", "etag": "etag-abc123"}`),
				},
			},
			tfIDs:     map[string]string{"resources.dashboards.my_dash": "d1"},
			wantKey:   "resources.dashboards.my_dash",
			wantID:    "d1",
			wantState: map[string]any{"etag": "etag-abc123"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := runBuildStateFromTF(t, tc.yaml, tc.tfAttrs, tc.tfIDs)

			if tc.absentKey != "" {
				assert.NotContains(t, state, tc.absentKey)
			}

			entry, ok := state[tc.wantKey]
			require.True(t, ok, "key %q not in state", tc.wantKey)
			assert.Equal(t, tc.wantID, entry.ID)

			if len(tc.wantState) > 0 {
				var got map[string]any
				require.NoError(t, json.Unmarshal(entry.State, &got))
				for k, v := range tc.wantState {
					assert.Equal(t, v, got[k], "state[%q]", k)
				}
			}

			if tc.wantStateRaw != "" {
				assert.Contains(t, string(entry.State), tc.wantStateRaw, "raw state JSON")
			}

			if tc.wantDeps != nil {
				require.Len(t, entry.DependsOn, len(tc.wantDeps))
				for i, want := range tc.wantDeps {
					assert.Equal(t, want.Node, entry.DependsOn[i].Node)
					if want.Label != "" {
						assert.Equal(t, want.Label, entry.DependsOn[i].Label)
					}
				}
			}
		})
	}
}
