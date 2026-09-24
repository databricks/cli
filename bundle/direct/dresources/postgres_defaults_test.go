package dresources

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPostgresTestClient(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "test-token",
	})
	require.NoError(t, err)
	return client
}

func createPostgresTestBranch(t *testing.T, client *databricks.WorkspaceClient, projectID string) string {
	t.Helper()
	ctx := t.Context()
	projectWaiter, err := client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{ProjectId: projectID})
	require.NoError(t, err)
	_, err = projectWaiter.Wait(ctx)
	require.NoError(t, err)

	branchName := "projects/" + projectID + "/branches/main"
	branchWaiter, err := client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
		Parent:   "projects/" + projectID,
		BranchId: "main",
	})
	require.NoError(t, err)
	_, err = branchWaiter.Wait(ctx)
	require.NoError(t, err)
	return branchName
}

func TestPostgresGeneratedRoleAndDatabaseIDs(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client := newPostgresTestClient(t, server)
	branchName := createPostgresTestBranch(t, client, "generated-ids")

	roleState := &PostgresRoleState{
		Parent: branchName,
		RoleRoleSpec: postgres.RoleRoleSpec{
			PostgresRole: "app_role",
		},
	}
	roleName, _, err := (&ResourcePostgresRole{}).New(client).DoCreate(t.Context(), roleState)
	require.NoError(t, err)
	assert.Regexp(t, `^projects/generated-ids/branches/main/roles/role-[a-f0-9]{8}$`, roleName)
	assert.Empty(t, roleState.RoleId, "create must keep the deployment ID out of leaf state")

	databaseState := &PostgresDatabaseState{
		Parent: branchName,
		DatabaseDatabaseSpec: postgres.DatabaseDatabaseSpec{
			PostgresDatabase: "app_db",
			Role:             roleName,
		},
	}
	databaseName, _, err := (&ResourcePostgresDatabase{}).New(client).DoCreate(t.Context(), databaseState)
	require.NoError(t, err)
	assert.Regexp(t, `^projects/generated-ids/branches/main/databases/db-[a-f0-9]{8}$`, databaseName)
	assert.Empty(t, databaseState.DatabaseId, "create must keep the deployment ID out of leaf state")
}

func TestPostgresProjectRemovalRestoresHistoryRetentionDefault(t *testing.T) {
	server := testserver.New(t)
	var updateBody map[string]any
	var updateMask string
	var callbackErr error
	server.RequestCallback = func(req *testserver.Request) {
		if req.Method == http.MethodPatch {
			callbackErr = json.Unmarshal(req.Body, &updateBody)
			updateMask = req.URL.Query().Get("update_mask")
		}
	}
	testserver.AddDefaultHandlers(server)
	client := newPostgresTestClient(t, server)
	ctx := t.Context()
	projectID := "retention-default"
	waiter, err := client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
		ProjectId: projectID,
		Project: postgres.Project{Spec: &postgres.ProjectSpec{
			HistoryRetentionDuration: duration.New(2 * 24 * time.Hour),
		}},
	})
	require.NoError(t, err)
	_, err = waiter.Wait(ctx)
	require.NoError(t, err)

	state := &PostgresProjectState{ProjectSpec: postgres.ProjectSpec{DisplayName: "updated"}}
	_, err = (&ResourcePostgresProject{}).New(client).DoUpdate(ctx, "projects/"+projectID, state, &PlanEntry{
		Changes: Changes{
			"history_retention_duration": {Action: deployplan.Update, Old: duration.New(2 * 24 * time.Hour), New: nil},
		},
	})
	require.NoError(t, err)
	require.NoError(t, callbackErr)
	assert.Nil(t, state.HistoryRetentionDuration, "wire default must not be copied into persisted desired state")
	assert.Equal(t, "604800s", updateBody["spec"].(map[string]any)["history_retention_duration"])
	assert.Equal(t, "spec.history_retention_duration", updateMask)

	project, err := client.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: "projects/" + projectID})
	require.NoError(t, err)
	require.NotNil(t, project.Status)
	require.NotNil(t, project.Status.HistoryRetentionDuration)
	assert.Equal(t, 7*24*time.Hour, project.Status.HistoryRetentionDuration.AsDuration())
}

func TestPostgresEndpointRemovalRestoresSuspendTimeoutDefault(t *testing.T) {
	server := testserver.New(t)
	var updateBody map[string]any
	var updateMask string
	var callbackErr error
	server.RequestCallback = func(req *testserver.Request) {
		if req.Method == http.MethodPatch {
			callbackErr = json.Unmarshal(req.Body, &updateBody)
			updateMask = req.URL.Query().Get("update_mask")
		}
	}
	testserver.AddDefaultHandlers(server)
	client := newPostgresTestClient(t, server)
	ctx := t.Context()
	branchName := createPostgresTestBranch(t, client, "suspend-default")
	endpointName := branchName + "/endpoints/read-only"
	waiter, err := client.Postgres.CreateEndpoint(ctx, postgres.CreateEndpointRequest{
		Parent:     branchName,
		EndpointId: "read-only",
		Endpoint: postgres.Endpoint{Spec: &postgres.EndpointSpec{
			EndpointType:           postgres.EndpointTypeEndpointTypeReadOnly,
			SuspendTimeoutDuration: duration.New(5 * time.Minute),
		}},
	})
	require.NoError(t, err)
	_, err = waiter.Wait(ctx)
	require.NoError(t, err)

	state := &PostgresEndpointState{EndpointSpec: postgres.EndpointSpec{EndpointType: postgres.EndpointTypeEndpointTypeReadOnly}}
	_, err = (&ResourcePostgresEndpoint{}).New(client).DoUpdate(ctx, endpointName, state, &PlanEntry{
		Changes: Changes{
			"suspend_timeout_duration": {Action: deployplan.Update, Old: duration.New(5 * time.Minute), New: nil},
		},
	})
	require.NoError(t, err)
	require.NoError(t, callbackErr)
	assert.Nil(t, state.SuspendTimeoutDuration, "wire default must not be copied into persisted desired state")
	assert.Equal(t, "spec.suspension", updateMask)
	assert.Equal(t, "86400s", updateBody["spec"].(map[string]any)["suspend_timeout_duration"])

	endpoint, err := client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: endpointName})
	require.NoError(t, err)
	require.NotNil(t, endpoint.Status)
	require.NotNil(t, endpoint.Status.SuspendTimeoutDuration)
	assert.Equal(t, 24*time.Hour, endpoint.Status.SuspendTimeoutDuration.AsDuration())
}
