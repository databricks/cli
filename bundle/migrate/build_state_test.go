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

// runBuildStateFromTF is a helper that runs BuildStateFromTF, finalizes the
// state, then reloads it so callers can inspect ResourceEntry (State + DependsOn).
func runBuildStateFromTF(
	t *testing.T,
	yaml string,
	tfAttrs migrate.TFStateAttrs,
	tfIDs terraform.ExportedResourcesMap,
	etags map[string]string,
) map[string]dstate.ResourceEntry {
	t.Helper()

	root := rootFromYAML(t, yaml)
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	statePath := filepath.Join(t.TempDir(), "resources.json")

	var db dstate.DeploymentState
	db.OpenWithData(statePath, dstate.NewDatabase("lineage", 1))
	require.NoError(t, db.UpgradeToWrite())

	err = migrate.BuildStateFromTF(t.Context(), &root, adapters, &db, tfAttrs, tfIDs, etags)
	require.NoError(t, err)

	_, err = db.Finalize(t.Context())
	require.NoError(t, err)

	// Reload from disk to access the full ResourceEntry (State JSON + DependsOn).
	raw, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var data dstate.Database
	require.NoError(t, json.Unmarshal(raw, &data))
	return data.State
}

func TestBuildStateFromTF_BasicJob(t *testing.T) {
	bundleYAML := `
resources:
  jobs:
    my_job:
      name: "hello"
`
	tfAttrs := migrate.TFStateAttrs{
		"databricks_job": {
			"my_job": json.RawMessage(`{"id": "123", "name": "hello"}`),
		},
	}
	tfIDs := terraform.ExportedResourcesMap{
		"resources.jobs.my_job": {ID: "123"},
	}

	state := runBuildStateFromTF(t, bundleYAML, tfAttrs, tfIDs, nil)
	entry, ok := state["resources.jobs.my_job"]
	require.True(t, ok)
	assert.Equal(t, "123", entry.ID)
	assert.Empty(t, entry.DependsOn)
}

func TestBuildStateFromTF_ResourceNotInTFState_Skipped(t *testing.T) {
	bundleYAML := `
resources:
  jobs:
    new_job:
      name: "new"
    existing_job:
      name: "existing"
`
	tfAttrs := migrate.TFStateAttrs{
		"databricks_job": {
			"existing_job": json.RawMessage(`{"id": "456", "name": "existing"}`),
		},
	}
	tfIDs := terraform.ExportedResourcesMap{
		"resources.jobs.existing_job": {ID: "456"},
	}

	state := runBuildStateFromTF(t, bundleYAML, tfAttrs, tfIDs, nil)
	assert.Contains(t, state, "resources.jobs.existing_job")
	assert.NotContains(t, state, "resources.jobs.new_job")
}

func TestBuildStateFromTF_DependsOnComputedFromRefs(t *testing.T) {
	bundleYAML := `
resources:
  pipelines:
    src:
      name: "source-pipeline"
  jobs:
    dst:
      name: "${resources.pipelines.src.name}"
`
	tfAttrs := migrate.TFStateAttrs{
		"databricks_pipeline": {
			"src": json.RawMessage(`{"id": "p1", "name": "source-pipeline"}`),
		},
		"databricks_job": {
			"dst": json.RawMessage(`{"id": "j1", "name": "source-pipeline"}`),
		},
	}
	tfIDs := terraform.ExportedResourcesMap{
		"resources.pipelines.src": {ID: "p1"},
		"resources.jobs.dst":      {ID: "j1"},
	}

	state := runBuildStateFromTF(t, bundleYAML, tfAttrs, tfIDs, nil)
	entry, ok := state["resources.jobs.dst"]
	require.True(t, ok)

	// depends_on must point back to the referenced pipeline
	require.Len(t, entry.DependsOn, 1)
	assert.Equal(t, deployplan.DependsOnEntry{
		Node:  "resources.pipelines.src",
		Label: "${resources.pipelines.src.name}",
	}, entry.DependsOn[0])

	// resolved field value
	var jobState map[string]any
	require.NoError(t, json.Unmarshal(entry.State, &jobState))
	assert.Equal(t, "source-pipeline", jobState["name"])
}

func TestBuildStateFromTF_NumericFieldReference(t *testing.T) {
	// dst_job.max_concurrent_runs references src_job's int field.
	// Verifies that the resolved value is stored as a number (not a string)
	// and that depends_on is recorded.
	bundleYAML := `
resources:
  jobs:
    src_job:
      name: "source"
      max_concurrent_runs: 4
    dst_job:
      name: "dest"
      max_concurrent_runs: "${resources.jobs.src_job.max_concurrent_runs}"
`
	tfAttrs := migrate.TFStateAttrs{
		"databricks_job": {
			"src_job": json.RawMessage(`{"id": "111", "name": "source", "max_concurrent_runs": 4}`),
			"dst_job": json.RawMessage(`{"id": "222", "name": "dest",   "max_concurrent_runs": 4}`),
		},
	}
	tfIDs := terraform.ExportedResourcesMap{
		"resources.jobs.src_job": {ID: "111"},
		"resources.jobs.dst_job": {ID: "222"},
	}

	state := runBuildStateFromTF(t, bundleYAML, tfAttrs, tfIDs, nil)

	entry, ok := state["resources.jobs.dst_job"]
	require.True(t, ok)

	// depends_on must point to src_job
	require.Len(t, entry.DependsOn, 1)
	assert.Equal(t, "resources.jobs.src_job", entry.DependsOn[0].Node)

	// max_concurrent_runs must be stored as a number, not a string
	var jobState map[string]any
	require.NoError(t, json.Unmarshal(entry.State, &jobState))
	assert.EqualValues(t, 4, jobState["max_concurrent_runs"])
}

func TestBuildStateFromTF_EtagStoredForDashboard(t *testing.T) {
	bundleYAML := `
resources:
  dashboards:
    my_dash:
      display_name: "My Dashboard"
`
	tfAttrs := migrate.TFStateAttrs{
		"databricks_dashboard": {
			"my_dash": json.RawMessage(`{"id": "d1", "display_name": "My Dashboard"}`),
		},
	}
	tfIDs := terraform.ExportedResourcesMap{
		"resources.dashboards.my_dash": {ID: "d1"},
	}
	etags := map[string]string{
		"resources.dashboards.my_dash": "etag-abc123",
	}

	state := runBuildStateFromTF(t, bundleYAML, tfAttrs, tfIDs, etags)
	entry, ok := state["resources.dashboards.my_dash"]
	require.True(t, ok)

	var dashState map[string]any
	require.NoError(t, json.Unmarshal(entry.State, &dashState))
	assert.Equal(t, "etag-abc123", dashState["etag"])
}
