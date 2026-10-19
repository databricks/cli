package phases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateToDMSAlreadyRecorded(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/export", func(req testserver.Request) any {
		return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST"}}
	})
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	b := &bundle.Bundle{
		BundleRootPath:    t.TempDir(),
		MigratingToDirect: true,
		Config: config.Root{
			Bundle: config.Bundle{
				Target:     "default",
				Deployment: config.Deployment{Lock: config.Lock{Enabled: new(false)}},
			},
			Workspace: config.Workspace{StatePath: "/Workspace/state"},
		},
	}
	b.SetWorkpaceClient(client)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	_, statePath := b.StateFilenameDirect(ctx)
	content := []byte(`{"state_version":3,"serial":2,"lineage":"lineage","features":{"deployment_history":{}},"state":{}}`)
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0o700))
	require.NoError(t, os.WriteFile(statePath, content, 0o600))
	b.DeploymentBundle.StateDB.OpenWithData(statePath, dstate.NewDatabase("lineage", 1))

	state := phases.MigrateToDMS(ctx, b)

	require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	require.NotNil(t, state)
	assert.True(t, state.IsDMS())
	assert.Equal(t, 2, state.Serial)
	assert.False(t, b.MigratingToDirect)
	assert.False(t, b.DeploymentBundle.StateDB.IsOpen())
	after, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, content, after)
}
