package phases_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindResolvesStableParentOutputs(t *testing.T) {
	for _, parentAction := range []deployplan.ActionType{deployplan.Skip, deployplan.Update} {
		for _, managedChild := range []bool{false, true} {
			for _, parent := range []string{"projects/p", "${resources.postgres_projects.project.name}"} {
				t.Run(fmt.Sprintf("%s/managed=%t/%s", parentAction, managedChild, parent), func(t *testing.T) {
					server := testserver.New(t)
					testserver.AddDefaultHandlers(server)
					client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
					require.NoError(t, err)
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
					if parentAction == deployplan.Update {
						source = strings.ReplaceAll(source, "existing project", "updated project")
					}
					if managedChild {
						source = strings.ReplaceAll(source, `,"lifecycle":{"bind":{"id":"projects/p/branches/b"}}`, "")
					}
					root, diagnostics := config.LoadFromBytes("databricks.yml", []byte(strings.ReplaceAll(source, "PARENT", parent)))
					require.Empty(t, diagnostics)
					ctx := logdiag.InitContext(t.Context())
					logdiag.SetCollect(ctx, true)
					configured := &bundle.Bundle{Config: *root}
					configured.SetWorkpaceClient(client)
					deployment := &configured.DeploymentBundle
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
					plan := phases.RunPlan(ctx, configured)
					require.NotNil(t, plan, "%v", logdiag.FlushCollected(ctx))
					assert.Empty(t, deployment.StateDB.GetResourceID("resources.postgres_projects.project"))
					assert.Equal(t, "projects/p", plan.Bind["resources.postgres_projects.project"].ID)
					assert.Equal(t, parentAction, plan.Plan["resources.postgres_projects.project"].Action)
					assert.Equal(t, deployplan.Skip, plan.Plan["resources.postgres_branches.branch"].Action)
				})
			}
		}
	}
}
