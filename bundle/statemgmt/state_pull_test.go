package statemgmt_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullResourcesStateRemoteRecordedCheckpoint(t *testing.T) {
	const recordedMarker = `{"state_version":3,"cli_version":"0.1.0","lineage":"recorded-lineage","serial":2,"features":{"deployment_history":{}},"state":{}}`
	const directSnapshot = `{"state_version":3,"lineage":"direct-lineage","serial":99,"state":{"resources.jobs.foo":{"__id__":"456","state":{"name":"foo"}}}}`
	const statePath = "/Workspace/state"
	tests := []struct {
		name       string
		configured *bool
	}{
		{name: "removed"},
		{name: "disabled", configured: new(false)},
		{name: "enabled", configured: new(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)
			b := &bundle.Bundle{
				BundleRootPath: t.TempDir(),
				Config: config.Root{
					Bundle:       config.Bundle{Target: "test"},
					Workspace:    config.Workspace{StatePath: statePath},
					Experimental: &config.Experimental{DeploymentHistory: tt.configured},
				},
			}
			b.SetWorkpaceClient(client)
			ctx := logdiag.InitContext(t.Context())
			logdiag.SetCollect(ctx, true)
			_, localPath := b.StateFilenameDirect(ctx)
			writeLocalState(t, localPath, directSnapshot)
			_, terraformPath := b.StateFilenameTerraform(ctx)
			writeLocalState(t, terraformPath, "invalid JSON")
			server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
				requestedPath := req.URL.Query().Get("path")
				if requestedPath == path.Join(statePath, "resources.json") {
					return workspace.ObjectInfo{Path: requestedPath, ObjectType: workspace.ObjectTypeFile}
				}
				return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST"}}
			})
			server.Handle("GET", "/api/2.0/workspace/export", func(req testserver.Request) any {
				return []byte(recordedMarker)
			})

			state := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true))
			require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
			require.NotNil(t, state)
			assert.True(t, state.IsDMS())
			assert.False(t, state.IsLocal)
			assert.Equal(t, 2, state.Serial)
			assert.Equal(t, "recorded-lineage", state.Lineage)
			content, err := os.ReadFile(localPath)
			require.NoError(t, err)
			assert.JSONEq(t, recordedMarker, string(content))
		})
	}
}

// writeLocalState prepares a local snapshot without running a deployment.
func writeLocalState(t *testing.T, filename, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o700))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o600))
}
