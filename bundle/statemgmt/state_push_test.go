package statemgmt_test

import (
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushResourcesStateMarkerWrites(t *testing.T) {
	const recordedMarker = `{"state_version":3,"cli_version":"0.1.0","lineage":"recorded-lineage","serial":2,"features":{"deployment_history":{}},"state":{}}`
	const directSnapshot = `{"state_version":3,"lineage":"direct-lineage","serial":99,"state":{"resources.jobs.foo":{"__id__":"456","state":{"name":"foo"}}}}`
	tests := []struct {
		name       string
		recorded   bool
		needsWrite bool
		missing    bool
		failUpload bool
		wantWrites int
		wantFlag   bool
	}{
		{name: "direct state always uploads", wantWrites: 2},
		{name: "direct upload does not consume migration marker", needsWrite: true, wantWrites: 2, wantFlag: true},
		{name: "recorded state skips ordinary uploads", recorded: true},
		{name: "recorded marker uploads once", recorded: true, needsWrite: true, wantWrites: 1},
		{name: "missing local marker keeps upload pending", recorded: true, needsWrite: true, missing: true, wantFlag: true},
		{name: "failed upload keeps marker pending", recorded: true, needsWrite: true, failUpload: true, wantWrites: 2, wantFlag: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)
			b := &bundle.Bundle{
				BundleRootPath: t.TempDir(),
				Config: config.Root{
					Bundle:    config.Bundle{Target: "test"},
					Workspace: config.Workspace{StatePath: "/Workspace/state"},
				},
			}
			b.SetWorkpaceClient(client)
			ctx := logdiag.InitContext(t.Context())
			logdiag.SetCollect(ctx, true)
			_, localPath := b.StateFilenameDirect(ctx)
			content := directSnapshot
			if tt.recorded {
				content = recordedMarker
			}
			writeLocalState(t, localPath, content)
			require.NoError(t, b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(tt.recorded), dstate.OpenDmsArgs{}))
			_, err = b.DeploymentBundle.StateDB.Finalize(ctx)
			require.NoError(t, err)
			if tt.missing {
				require.NoError(t, os.Remove(localPath))
			}
			b.NeedsDMSStateUpload = tt.needsWrite
			writes := 0
			server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
				writes++
				if tt.failUpload {
					return testserver.Response{StatusCode: 403, Body: map[string]string{"error_code": "PERMISSION_DENIED", "message": "access denied"}}
				}
				return map[string]any{}
			})
			statemgmt.PushResourcesState(ctx, b)
			statemgmt.PushResourcesState(ctx, b)
			assert.Equal(t, tt.wantWrites, writes)
			assert.Equal(t, tt.wantFlag, b.NeedsDMSStateUpload)
			assert.Equal(t, tt.failUpload, logdiag.HasError(ctx))
		})
	}
}
