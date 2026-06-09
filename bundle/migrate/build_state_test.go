package migrate_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/terraform"
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
	tfIDs terraform.ExportedResourcesMap,
) map[string]dstate.ResourceEntry {
	t.Helper()

	root := rootFromYAML(t, yaml)
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	statePath := filepath.Join(t.TempDir(), "resources.json")

	var db dstate.DeploymentState
	db.OpenWithData(statePath, dstate.NewDatabase("lineage", 1))
	require.NoError(t, db.UpgradeToWrite())

	err = migrate.BuildStateFromTF(t.Context(), &root, adapters, &db, tfAttrs, tfIDs)
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
		name      string
		yaml      string
		tfAttrs   migrate.TFStateAttrs
		tfIDs     terraform.ExportedResourcesMap
		wantKey   string         // primary key to assert on
		absentKey string         // key that must NOT be in state
		wantID    string         // expected entry.ID
		wantState map[string]any // expected fields in the state JSON
		wantDeps  []deployplan.DependsOnEntry
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
			tfIDs:   terraform.ExportedResourcesMap{"resources.jobs.my_job": {ID: "123"}},
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
			tfIDs:     terraform.ExportedResourcesMap{"resources.jobs.existing_job": {ID: "456"}},
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
			tfIDs: terraform.ExportedResourcesMap{
				"resources.pipelines.src": {ID: "p1"},
				"resources.jobs.dst":      {ID: "j1"},
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
			tfIDs: terraform.ExportedResourcesMap{
				"resources.jobs.src_job": {ID: "111"},
				"resources.jobs.dst_job": {ID: "222"},
			},
			wantKey:   "resources.jobs.dst_job",
			wantID:    "222",
			wantState: map[string]any{"max_concurrent_runs": float64(4)}, // JSON numbers decode as float64
			wantDeps:  []deployplan.DependsOnEntry{{Node: "resources.jobs.src_job"}},
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
			tfIDs:     terraform.ExportedResourcesMap{"resources.dashboards.my_dash": {ID: "d1"}},
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
