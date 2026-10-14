package statemgmt_test

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	recordedMarker = `{"state_version":3,"cli_version":"0.1.0","lineage":"recorded-lineage","serial":2,"features":{"deployment_history":{}},"state":{}}`
	directSnapshot = `{"state_version":3,"lineage":"direct-lineage","serial":99,"state":{"resources.jobs.foo":{"__id__":"456","state":{"name":"foo"}}}}`
	statePath      = "/Workspace/state"
)

// newStateTestBundle isolates local files and diagnostics for state I/O tests.
func newStateTestBundle(t *testing.T) (context.Context, *bundle.Bundle, *testserver.Server) {
	t.Helper()
	server := testserver.New(t)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle:       config.Bundle{Target: "test"},
			Workspace:    config.Workspace{StatePath: statePath},
			Experimental: &config.Experimental{DeploymentHistory: new(false)},
		},
	}
	b.SetWorkpaceClient(client)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	return ctx, b, server
}

func TestPullResourcesStateRepair(t *testing.T) {
	tests := []struct {
		name           string
		remote         string
		local          string
		node           bool
		historyEnabled bool
		noLookup       bool
		remoteError    bool
		recordError    bool
		wal            bool
		wantErr        string
		wantRepair     bool
		wantRecorded   bool
		wantNew        bool
	}{
		{name: "new direct deployment", noLookup: true},
		{name: "local recorded marker without backend", local: recordedMarker, wantRecorded: true},
		{name: "syntax error", remote: "{", node: true, wantRepair: true, wantNew: true},
		{name: "header type error", remote: `{"serial":"invalid"}`, node: true, wantRepair: true, wantNew: true},
		{name: "resource type error", remote: `{"state":[]}`, node: true, wantRepair: true, wantNew: true},
		{name: "missing marker", node: true, historyEnabled: true, wantRepair: true, wantNew: true},
		{name: "surviving recorded header", remote: "{", local: recordedMarker, node: true, wantRepair: true},
		{name: "missing marker with surviving header", local: recordedMarker, node: true, wantRepair: true},
		{name: "both copies malformed", remote: "{", local: "{", node: true, wantRepair: true, wantNew: true},
		{name: "corrupt direct without deployment", remote: "{", wantErr: "parsing state"},
		{name: "local corruption without deployment", local: "{", wantErr: "parsing"},
		{name: "node without record", remote: "{", node: true, recordError: true, wantErr: "record not found"},
		{name: "remote permission failure", node: true, remoteError: true, wantErr: "access denied"},
		{name: "valid remote direct snapshot", remote: directSnapshot, node: true},
		{name: "valid local direct snapshot", local: directSnapshot, node: true},
		{name: "corrupt remote with direct local resources", remote: "{", local: directSnapshot, node: true, wantErr: "parsing state"},
		{name: "newer surviving schema", remote: "{", local: strings.Replace(recordedMarker, `"state_version":3`, `"state_version":99`, 1), node: true, wantErr: "newer than supported"},
		{name: "unknown surviving feature", remote: "{", local: strings.Replace(recordedMarker, `"deployment_history":{}`, `"deployment_history":{},"future_feature":{}`, 1), node: true, wantErr: "features this CLI does not support"},
		{name: "surviving WAL", remote: "{", local: recordedMarker, node: true, wal: true, wantErr: "unexpected WAL"},
		{name: "surviving resources are not a marker", remote: "{", local: strings.Replace(directSnapshot, `"serial":99`, `"serial":99,"features":{"deployment_history":{}}`, 1), node: true, wantErr: "expected none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, b, server := newStateTestBundle(t)
			b.Config.Experimental.DeploymentHistory = new(tt.historyEnabled)
			_, localPath := b.StateFilenameDirect(ctx)
			if tt.local != "" {
				writeLocalState(t, localPath, tt.local)
			}
			if tt.wal {
				writeLocalState(t, localPath+".wal", "{}\n")
			}
			server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
				requestedPath := req.URL.Query().Get("path")
				switch requestedPath {
				case path.Join(statePath, "resources.json"):
					if tt.remoteError {
						return testserver.Response{StatusCode: 403, Body: map[string]string{"error_code": "PERMISSION_DENIED", "message": "access denied"}}
					}
					if tt.remote != "" {
						return workspace.ObjectInfo{Path: requestedPath, ObjectType: workspace.ObjectTypeFile}
					}
				case path.Join(statePath, dms.DeploymentNodeName):
					assert.False(t, tt.noLookup, "fresh direct state must not query DMS")
					if tt.node {
						return workspace.ObjectInfo{Path: requestedPath, ObjectId: 123}
					}
				}
				return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "not found"}}
			})
			server.Handle("GET", "/api/2.0/workspace/export", func(req testserver.Request) any {
				return []byte(tt.remote)
			})
			server.Handle("GET", "/api/2.0/bundle/deployments/123", func(req testserver.Request) any {
				if tt.recordError {
					return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "record not found"}}
				}
				return bundledeployments.Deployment{Name: dms.DeploymentName("123"), LastVersionId: "7"}
			})

			state := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true))
			assert.Equal(t, tt.wantRepair, b.NeedsDMSStateUpload)
			if tt.wantErr != "" {
				require.True(t, logdiag.HasError(ctx))
				assert.Contains(t, logdiag.GetFirstErrorSummary(ctx), tt.wantErr)
				return
			}
			require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
			require.NotNil(t, state)
			if !tt.wantRepair {
				assert.Equal(t, tt.wantRecorded, state.IsDMS())
				return
			}
			assert.True(t, state.IsDMS())
			assert.Equal(t, "123", state.DeploymentID)
			require.NotNil(t, state.Deployment)
			assert.Equal(t, "7", state.Deployment.LastVersionId)
			content, err := os.ReadFile(localPath)
			require.NoError(t, err)
			var reconstructed dstate.Database
			require.NoError(t, json.Unmarshal(content, &reconstructed))
			assert.Empty(t, reconstructed.State)
			if tt.wantNew {
				assert.NotEmpty(t, state.Lineage)
				assert.NotEqual(t, "recorded-lineage", state.Lineage)
				assert.Equal(t, 1, logdiag.NumWarnings(ctx))
			} else {
				assert.Equal(t, "recorded-lineage", state.Lineage)
				assert.Equal(t, 2, state.Serial)
				assert.Equal(t, "0.1.0", reconstructed.CLIVersion)
				assert.Zero(t, logdiag.NumWarnings(ctx))
			}
		})
	}
}
