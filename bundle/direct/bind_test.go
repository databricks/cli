package direct_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBindPreparesIsolatedState checks that bind resolves a preview without committing it.
func TestBindPreparesIsolatedState(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	statePath := filepath.Join(t.TempDir(), "resources.json")
	// Start with an already-managed job, "kept" (ID 10). Its state must stay unchanged.
	data := dstate.NewDatabase("lineage", 4)
	data.Features = map[string]struct{}{dstate.FeatureDeploymentHistoryShadow: {}}
	data.State["resources.jobs.kept"] = dstate.ResourceEntry{
		ID: "10", State: json.RawMessage(`{"name":"kept"}`),
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.old"}},
	}
	original, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, original, 0o600))

	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/get", func(req testserver.Request) any {
		// The planner reads these jobs while preparing the preview. Check the files here
		// so a temporary write cannot go unnoticed just because bind deletes it later.
		assert.NoFileExists(t, statePath+".wal")
		assert.NoFileExists(t, statePath+".temp-bind")
		actual, err := os.ReadFile(statePath)
		require.NoError(t, err)
		assert.Equal(t, original, actual)
		if req.URL.Query().Get("job_id") == "10" {
			return jobs.Job{JobId: 10, Settings: &jobs.JobSettings{Name: "kept"}}
		}
		return jobs.Job{JobId: 20, Settings: &jobs.JobSettings{Name: "remote"}}
	})
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	// Bind "bound" to job 20. Its tag references kept's ID, so the preview must resolve
	// that reference while leaving the real state and caches alone.
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`resources:
  jobs:
    kept:
      name: kept
    bound:
      name: configured
      tags:
        other: ${resources.jobs.kept.id}
`))
	require.Empty(t, diags)
	var b direct.DeploymentBundle
	// Seed the real caches so using them for the preview would be detectable.
	cached := structvar.NewStructVar(&jobs.JobSettings{Name: "original cache"}, nil)
	b.StateCache.Store("resources.jobs.bound", cached)
	b.RemoteStateCache.Store("resources.jobs.bound", "original remote cache")

	prepared, err := b.Bind(ctx, client, root, statePath, "resources.jobs.bound", "20")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := b.StateDB.Finalize(ctx)
		require.NoError(t, err)
	})
	require.False(t, logdiag.HasError(ctx))

	// Before approval, real state still contains only "kept". The proposed "bound"
	// entry belongs only to the preview, and no serial or existing state has changed.
	assert.Equal(t, data, b.StateDB.Data)
	assert.Empty(t, b.StateDB.GetResourceID("resources.jobs.bound"))
	assert.Equal(t, "10", b.StateDB.GetResourceID("resources.jobs.kept"))

	// The preview must also leave the real plan and seeded cache entries untouched.
	assert.Nil(t, b.Plan)
	actualCache, ok := b.StateCache.Load("resources.jobs.bound")
	require.True(t, ok)
	assert.Same(t, cached, actualCache)
	remoteCache, ok := b.RemoteStateCache.Load("resources.jobs.bound")
	require.True(t, ok)
	assert.Equal(t, "original remote cache", remoteCache)

	// The prepared binding keeps job 20's ID, resolves kept's ID to 10, and records
	// the dependency for the state that will be saved after approval.
	assert.Equal(t, "20", prepared.Resource.ID)
	assert.JSONEq(t, `{"name":"configured","tags":{"other":"10"}}`, string(prepared.Resource.State))
	assert.Contains(t, prepared.Resource.DependsOn, deployplan.DependsOnEntry{Node: "resources.jobs.kept", Label: "${resources.jobs.kept.id}"})
	// A later deploy would change job 20's name from "remote" to "configured".
	assert.True(t, prepared.HasChanges)
	assert.Equal(t, deployplan.Update, prepared.Action)
}
