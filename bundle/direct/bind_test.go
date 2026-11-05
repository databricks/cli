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
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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
				adapter, err := dresources.NewAdapter(dresources.SupportedResources[resourceType], resourceType, client)
				require.NoError(t, err)
				unit := direct.DeploymentUnit{
					ResourceKey: "resources." + resourceType + ".test",
					Adapter:     adapter,
				}
				var newState any = &catalog.Secret{Name: "test", Value: secret, EffectiveValue: secret}
				rawContent := secret
				if resourceType == "dashboards" {
					newState = &dresources.DashboardState{
						SerializedDashboard: content,
					}
					rawContent = content
				}
				require.NoError(t, unit.Bind(ctx, &stateDB, "remote-id", newState, &deployplan.PlanEntry{}))
				assert.Equal(t, "remote-id", stateDB.GetResourceID(unit.ResourceKey))
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

func TestBindResolvesStableParentOutputs(t *testing.T) {
	for _, parentAction := range []deployplan.ActionType{deployplan.Bind, deployplan.BindAndUpdate} {
		for _, managedChild := range []bool{false, true} {
			for _, parent := range []string{"projects/p", "${resources.postgres_projects.project.name}"} {
				t.Run(fmt.Sprintf("%s/managed=%t/%s", parentAction, managedChild, parent), func(t *testing.T) {
					server, client := bindTestClient(t)
					server.Handle("GET", "/api/2.0/postgres/projects/p", func(req testserver.Request) any {
						return postgres.Project{
							Name:      "projects/p",
							ProjectId: "p",
							Spec:      &postgres.ProjectSpec{DisplayName: "existing project"},
						}
					})
					server.Handle("GET", "/api/2.0/postgres/projects/p/branches/b", func(req testserver.Request) any {
						return postgres.Branch{Name: "projects/p/branches/b", BranchId: "b", Parent: "projects/p"}
					})
					source := `{"resources":{"postgres_projects":{"project":{"project_id":"p","display_name":"existing project","lifecycle":{"bind":{"id":"projects/p"}}}},"postgres_branches":{"branch":{"parent":"PARENT","branch_id":"b","lifecycle":{"bind":{"id":"projects/p/branches/b"}}}}}}`
					if parentAction == deployplan.BindAndUpdate {
						source = strings.ReplaceAll(source, "existing project", "updated project")
					}
					if managedChild {
						source = strings.ReplaceAll(source, `,"lifecycle":{"bind":{"id":"projects/p/branches/b"}}`, "")
					}
					root, diagnostics := config.LoadFromBytes("databricks.yml", []byte(strings.ReplaceAll(source, "PARENT", parent)))
					require.Empty(t, diagnostics)
					ctx := logdiag.InitContext(t.Context())
					logdiag.SetCollect(ctx, true)
					var deployment direct.DeploymentBundle
					statePath := filepath.Join(t.TempDir(), "resources.json")
					if managedChild {
						require.NoError(t, deployment.StateDB.Open(ctx, statePath, dstate.WritingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}))
						require.NoError(t, deployment.StateDB.SaveState(ctx, "resources.postgres_branches.branch", "projects/p/branches/b", &resources.PostgresBranchConfig{Parent: "projects/p", BranchId: "b"}, nil))
						_, err := deployment.StateDB.Finalize(ctx)
						require.NoError(t, err)
					}
					require.NoError(t, deployment.StateDB.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}))
					t.Cleanup(func() {
						_, err := deployment.StateDB.Finalize(ctx)
						assert.NoError(t, err)
					})
					plan, err := deployment.CalculatePlan(ctx, client, root)
					require.NoError(t, err, "%v", logdiag.FlushCollected(ctx))
					assert.Equal(t, parentAction, plan.Plan["resources.postgres_projects.project"].Action)
					expected := deployplan.Bind
					if managedChild {
						expected = deployplan.Skip
					}
					assert.Equal(t, expected, plan.Plan["resources.postgres_branches.branch"].Action)
				})
			}
		}
	}
}
