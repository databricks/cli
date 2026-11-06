package direct_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bindTestClient(t *testing.T) (*testserver.Server, *databricks.WorkspaceClient) {
	t.Helper()
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	return server, client
}

func TestPureBindSanitizesPersistedState(t *testing.T) {
	const secret = "synthetic-sensitive-value"
	content := strings.Repeat("dashboard content ", 20)
	for _, mode := range []dstate.Mode{
		dstate.WritingFile,
		dstate.WritingDeploymentHistoryShadow,
		dstate.WritingDeploymentHistoryOnly,
	} {
		for _, resourceType := range []string{"secrets", "dashboards"} {
			t.Run(fmt.Sprintf("%d/%s", mode, resourceType), func(t *testing.T) {
				server, client := bindTestClient(t)
				recorded := make(chan string, 1)
				server.Handle("PATCH", "/api/2.0/bundle/deployments/123/versions/1/operations/{key}", func(req testserver.Request) any {
					var operation bundledeployments.Operation
					assert.NoError(t, json.Unmarshal(req.Body, &operation))
					recorded <- operation.State
					return bundledeployments.Operation{SequenceId: 1}
				})
				ctx := cmdctx.SetWorkspaceClient(t.Context(), client)
				ctx = logdiag.InitContext(ctx)
				statePath := filepath.Join(t.TempDir(), "resources.json")
				var stateDB dstate.DeploymentState
				require.NoError(t, stateDB.Open(ctx, statePath, mode, dstate.WithRecovery(true), dstate.OpenDmsArgs{}))
				t.Cleanup(func() {
					_, err := stateDB.Finalize(ctx)
					assert.NoError(t, err)
				})
				if mode != dstate.WritingFile {
					stateDB.InitializeOperationBuffer(ctx, "123", 1, 1)
				}
				resourceKey := "resources." + resourceType + ".test"
				var newState any = &catalog.Secret{Name: "test", Value: secret, EffectiveValue: secret}
				server.Handle("GET", "/api/2.1/unity-catalog/secrets/remote-id", func(req testserver.Request) any {
					return &catalog.Secret{Name: "test", EffectiveValue: secret}
				})
				server.Handle("GET", "/api/2.0/lakeview/dashboards/remote-id", func(req testserver.Request) any {
					return &dashboards.Dashboard{DashboardId: "remote-id", SerializedDashboard: content}
				})
				server.Handle("GET", "/api/2.0/lakeview/dashboards/remote-id/published", func(req testserver.Request) any {
					return &dashboards.PublishedDashboard{}
				})
				rawContent := secret
				if resourceType == "dashboards" {
					newState = &dresources.DashboardState{
						SerializedDashboard: content,
					}
					rawContent = content
				}
				definitionJSON, err := json.Marshal(newState)
				require.NoError(t, err)
				var definition map[string]any
				require.NoError(t, json.Unmarshal(definitionJSON, &definition))
				delete(definition, "effective_value")
				delete(definition, "published")
				definition["lifecycle"] = map[string]any{"bind": map[string]string{"id": "remote-id"}}
				configJSON, err := json.Marshal(map[string]any{"resources": map[string]any{resourceType: map[string]any{"test": definition}}})
				require.NoError(t, err)
				root, diagnostics := config.LoadFromBytes("databricks.yml", configJSON)
				require.Empty(t, diagnostics)
				var deployment direct.DeploymentBundle
				deployment.StateDB.OpenWithData(statePath, dstate.NewDatabase("", 0))
				bindings, err := deployment.PrepareBindings(ctx, client, root, nil)
				require.NoError(t, err)
				require.NoError(t, stateDB.Bind(ctx, bindings))
				assert.Equal(t, "remote-id", stateDB.GetResourceID(resourceKey))
				if mode != dstate.WritingDeploymentHistoryOnly {
					wal, err := os.ReadFile(statePath + ".wal")
					require.NoError(t, err)
					assert.NotContains(t, string(wal), rawContent)
					if resourceType == "dashboards" {
						assert.Contains(t, string(wal), fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content))))
					}
				}
				_, err = stateDB.Finalize(ctx)
				require.NoError(t, err)
				persisted, err := os.ReadFile(statePath)
				require.NoError(t, err)
				assert.NotContains(t, string(persisted), rawContent)
				if mode != dstate.WritingFile {
					require.Len(t, recorded, 1)
					history := <-recorded
					assert.NotContains(t, history, rawContent)
					assert.Contains(t, history, `"state":`)
					if resourceType == "dashboards" {
						assert.Contains(t, history, fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content))))
					}
				}
			})
		}
	}
}
