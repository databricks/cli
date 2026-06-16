package testserver_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresProjectCRUD(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project
	createReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=test-project", nil)
	createReq.Header.Set("Authorization", "Bearer test-token")
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := client.Do(createReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createResp.StatusCode)

	var createOp postgres.Operation
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&createOp))
	assert.True(t, createOp.Done)
	createResp.Body.Close()

	// Get project
	getReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/test-project", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getResp.StatusCode)

	var project postgres.Project
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&project))
	assert.Equal(t, "projects/test-project", project.Name)
	getResp.Body.Close()

	// List projects
	listReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects", nil)
	listReq.Header.Set("Authorization", "Bearer test-token")
	listResp, err := client.Do(listReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listResp.StatusCode)

	var listProjects postgres.ListProjectsResponse
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listProjects))
	assert.Len(t, listProjects.Projects, 1)
	listResp.Body.Close()

	// Delete project
	deleteReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/2.0/postgres/projects/test-project", nil)
	deleteReq.Header.Set("Authorization", "Bearer test-token")
	deleteResp, err := client.Do(deleteReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteResp.StatusCode)
	deleteResp.Body.Close()

	// Verify project is deleted
	getReq2, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/test-project", nil)
	getReq2.Header.Set("Authorization", "Bearer test-token")
	getResp2, err := client.Do(getReq2)
	require.NoError(t, err)
	assert.Equal(t, 404, getResp2.StatusCode)
	getResp2.Body.Close()
}

func TestPostgresProjectNotFound(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	getReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/nonexistent", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	assert.Equal(t, 404, getResp.StatusCode)
	getResp.Body.Close()
}

func TestPostgresProjectDuplicate(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project
	createReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=dup-project", nil)
	createReq.Header.Set("Authorization", "Bearer test-token")
	createResp, err := client.Do(createReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createResp.StatusCode)
	createResp.Body.Close()

	// Try to create duplicate
	createReq2, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=dup-project", nil)
	createReq2.Header.Set("Authorization", "Bearer test-token")
	createResp2, err := client.Do(createReq2)
	require.NoError(t, err)
	assert.Equal(t, 409, createResp2.StatusCode)
	createResp2.Body.Close()
}

func TestPostgresBranchCRUD(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=branch-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/branch-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Get branch
	getBranchReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/branch-test-project/branches/main", nil)
	getBranchReq.Header.Set("Authorization", "Bearer test-token")
	getBranchResp, err := client.Do(getBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getBranchResp.StatusCode)

	var branch postgres.Branch
	require.NoError(t, json.NewDecoder(getBranchResp.Body).Decode(&branch))
	assert.Equal(t, "projects/branch-test-project/branches/main", branch.Name)
	getBranchResp.Body.Close()

	// List branches
	listBranchReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/branch-test-project/branches", nil)
	listBranchReq.Header.Set("Authorization", "Bearer test-token")
	listBranchResp, err := client.Do(listBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listBranchResp.StatusCode)

	var listBranches postgres.ListBranchesResponse
	require.NoError(t, json.NewDecoder(listBranchResp.Body).Decode(&listBranches))
	// Expect 2 branches: the default branch created with the project + "main" branch
	assert.Len(t, listBranches.Branches, 2)
	listBranchResp.Body.Close()

	// Delete branch
	deleteBranchReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/2.0/postgres/projects/branch-test-project/branches/main", nil)
	deleteBranchReq.Header.Set("Authorization", "Bearer test-token")
	deleteBranchResp, err := client.Do(deleteBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteBranchResp.StatusCode)
	deleteBranchResp.Body.Close()
}

func TestPostgresBranchNotFoundWhenProjectNotExists(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Try to create branch without project
	createBranchReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/nonexistent/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 404, createBranchResp.StatusCode)
	createBranchResp.Body.Close()
}

func TestPostgresEndpointCRUD(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=endpoint-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Create endpoint
	createEpReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints?endpoint_id=rw-endpoint", nil)
	createEpReq.Header.Set("Authorization", "Bearer test-token")
	createEpResp, err := client.Do(createEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createEpResp.StatusCode)
	createEpResp.Body.Close()

	// Get endpoint
	getEpReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", nil)
	getEpReq.Header.Set("Authorization", "Bearer test-token")
	getEpResp, err := client.Do(getEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getEpResp.StatusCode)

	var endpoint postgres.Endpoint
	require.NoError(t, json.NewDecoder(getEpResp.Body).Decode(&endpoint))
	assert.Equal(t, "projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", endpoint.Name)
	getEpResp.Body.Close()

	// List endpoints
	listEpReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints", nil)
	listEpReq.Header.Set("Authorization", "Bearer test-token")
	listEpResp, err := client.Do(listEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listEpResp.StatusCode)

	// Each branch implicitly provisions a "primary" RW endpoint in addition to
	// the explicitly-created "rw-endpoint".
	var listEndpoints postgres.ListEndpointsResponse
	require.NoError(t, json.NewDecoder(listEpResp.Body).Decode(&listEndpoints))
	assert.Len(t, listEndpoints.Endpoints, 2)
	listEpResp.Body.Close()

	// Delete endpoint
	deleteEpReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", nil)
	deleteEpReq.Header.Set("Authorization", "Bearer test-token")
	deleteEpResp, err := client.Do(deleteEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteEpResp.StatusCode)
	deleteEpResp.Body.Close()
}

func TestPostgresDatabaseCRUD(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=database-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/database-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Create database
	createDbBody := `{"spec":{"postgres_database":"my_db"}}`
	createDbReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases?database_id=my-db", strings.NewReader(createDbBody))
	createDbReq.Header.Set("Authorization", "Bearer test-token")
	createDbReq.Header.Set("Content-Type", "application/json")
	createDbResp, err := client.Do(createDbReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createDbResp.StatusCode)
	createDbResp.Body.Close()

	// Get database
	getDbReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases/my-db", nil)
	getDbReq.Header.Set("Authorization", "Bearer test-token")
	getDbResp, err := client.Do(getDbReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getDbResp.StatusCode)

	var database postgres.Database
	require.NoError(t, json.NewDecoder(getDbResp.Body).Decode(&database))
	assert.Equal(t, "projects/database-test-project/branches/main/databases/my-db", database.Name)
	assert.Equal(t, "projects/database-test-project/branches/main", database.Parent)
	require.NotNil(t, database.Status)
	assert.Equal(t, "my_db", database.Status.PostgresDatabase)
	assert.NotEmpty(t, database.Status.Role)
	getDbResp.Body.Close()

	// List databases
	listDbReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases", nil)
	listDbReq.Header.Set("Authorization", "Bearer test-token")
	listDbResp, err := client.Do(listDbReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listDbResp.StatusCode)

	var listDatabases postgres.ListDatabasesResponse
	require.NoError(t, json.NewDecoder(listDbResp.Body).Decode(&listDatabases))
	assert.Len(t, listDatabases.Databases, 1)
	listDbResp.Body.Close()

	// Update database (rename via spec.postgres_database)
	updateDbBody := `{"spec":{"postgres_database":"my_db_renamed"}}`
	updateDbReq, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases/my-db?update_mask=spec.postgres_database", strings.NewReader(updateDbBody))
	updateDbReq.Header.Set("Authorization", "Bearer test-token")
	updateDbReq.Header.Set("Content-Type", "application/json")
	updateDbResp, err := client.Do(updateDbReq)
	require.NoError(t, err)
	assert.Equal(t, 200, updateDbResp.StatusCode)
	updateDbResp.Body.Close()

	// Verify rename was applied
	getDbReq2, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases/my-db", nil)
	getDbReq2.Header.Set("Authorization", "Bearer test-token")
	getDbResp2, err := client.Do(getDbReq2)
	require.NoError(t, err)
	assert.Equal(t, 200, getDbResp2.StatusCode)
	var database2 postgres.Database
	require.NoError(t, json.NewDecoder(getDbResp2.Body).Decode(&database2))
	require.NotNil(t, database2.Status)
	assert.Equal(t, "my_db_renamed", database2.Status.PostgresDatabase)
	getDbResp2.Body.Close()

	// Delete database
	deleteDbReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/2.0/postgres/projects/database-test-project/branches/main/databases/my-db", nil)
	deleteDbReq.Header.Set("Authorization", "Bearer test-token")
	deleteDbResp, err := client.Do(deleteDbReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteDbResp.StatusCode)
	deleteDbResp.Body.Close()
}

func TestPostgresDatabaseNotFoundWhenBranchNotExists(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=db-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Try to create database without branch
	createDbReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/db-test-project/branches/nonexistent/databases?database_id=my-db", nil)
	createDbReq.Header.Set("Authorization", "Bearer test-token")
	createDbResp, err := client.Do(createDbReq)
	require.NoError(t, err)
	assert.Equal(t, 404, createDbResp.StatusCode)
	createDbResp.Body.Close()
}

func TestPostgresDatabaseUpdateMaskPreservesUnmaskedFields(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	do := func(method, url, body string) *http.Response {
		req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		return resp
	}

	do(http.MethodPost, "/api/2.0/postgres/projects?project_id=mask-db-project", "").Body.Close()
	do(http.MethodPost, "/api/2.0/postgres/projects/mask-db-project/branches?branch_id=main", "").Body.Close()

	createBody := `{"spec":{"postgres_database":"app_db","role":"projects/mask-db-project/branches/main/roles/owner"}}`
	createResp := do(http.MethodPost, "/api/2.0/postgres/projects/mask-db-project/branches/main/databases?database_id=appdb", createBody)
	require.Equal(t, 200, createResp.StatusCode)
	createResp.Body.Close()

	// Update only spec.postgres_database. The body also carries a different role,
	// which must be ignored because it is not named in update_mask.
	patchBody := `{"spec":{"postgres_database":"renamed_db","role":"projects/mask-db-project/branches/main/roles/other"}}`
	patchResp := do(http.MethodPatch, "/api/2.0/postgres/projects/mask-db-project/branches/main/databases/appdb?update_mask=spec.postgres_database", patchBody)
	assert.Equal(t, 200, patchResp.StatusCode)
	patchResp.Body.Close()

	getResp := do(http.MethodGet, "/api/2.0/postgres/projects/mask-db-project/branches/main/databases/appdb", "")
	var database postgres.Database
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&database))
	getResp.Body.Close()

	require.NotNil(t, database.Status)
	assert.Equal(t, "renamed_db", database.Status.PostgresDatabase, "masked field should be updated")
	assert.Equal(t, "projects/mask-db-project/branches/main/roles/owner", database.Status.Role, "field absent from update_mask should be preserved")
}

func TestPostgresDatabaseCreateDuplicateReturns400(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	do := func(method, url, body string) *http.Response {
		req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		return resp
	}

	do(http.MethodPost, "/api/2.0/postgres/projects?project_id=dup-db-project", "").Body.Close()
	do(http.MethodPost, "/api/2.0/postgres/projects/dup-db-project/branches?branch_id=main", "").Body.Close()

	createBody := `{"spec":{"postgres_database":"app_db"}}`
	first := do(http.MethodPost, "/api/2.0/postgres/projects/dup-db-project/branches/main/databases?database_id=appdb", createBody)
	require.Equal(t, 200, first.StatusCode)
	first.Body.Close()

	// Creating the same database again fails the way the real API does: 400, not 409.
	second := do(http.MethodPost, "/api/2.0/postgres/projects/dup-db-project/branches/main/databases?database_id=appdb", createBody)
	assert.Equal(t, 400, second.StatusCode)
	second.Body.Close()
}

func TestPostgresEndpointNotFoundWhenBranchNotExists(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=ep-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Try to create endpoint without branch
	createEpReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/ep-test-project/branches/nonexistent/endpoints?endpoint_id=rw-endpoint", nil)
	createEpReq.Header.Set("Authorization", "Bearer test-token")
	createEpResp, err := client.Do(createEpReq)
	require.NoError(t, err)
	assert.Equal(t, 404, createEpResp.StatusCode)
	createEpResp.Body.Close()
}

func TestPostgresRoleCRUD(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=role-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/role-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Create role
	createRoleBody := `{"spec":{"postgres_role":"my_role"}}`
	createRoleReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles?role_id=my-role", strings.NewReader(createRoleBody))
	createRoleReq.Header.Set("Authorization", "Bearer test-token")
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleResp, err := client.Do(createRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createRoleResp.StatusCode)
	createRoleResp.Body.Close()

	// Get role
	getRoleReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles/my-role", nil)
	getRoleReq.Header.Set("Authorization", "Bearer test-token")
	getRoleResp, err := client.Do(getRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getRoleResp.StatusCode)

	var role postgres.Role
	require.NoError(t, json.NewDecoder(getRoleResp.Body).Decode(&role))
	assert.Equal(t, "projects/role-test-project/branches/main/roles/my-role", role.Name)
	assert.Equal(t, "projects/role-test-project/branches/main", role.Parent)
	require.NotNil(t, role.Status)
	assert.Equal(t, "my_role", role.Status.PostgresRole)
	getRoleResp.Body.Close()

	// List roles
	listRoleReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles", nil)
	listRoleReq.Header.Set("Authorization", "Bearer test-token")
	listRoleResp, err := client.Do(listRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listRoleResp.StatusCode)

	var listRoles postgres.ListRolesResponse
	require.NoError(t, json.NewDecoder(listRoleResp.Body).Decode(&listRoles))
	assert.Len(t, listRoles.Roles, 1)
	listRoleResp.Body.Close()

	// Update role (rename via spec.postgres_role)
	updateRoleBody := `{"spec":{"postgres_role":"my_role_renamed"}}`
	updateRoleReq, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles/my-role", strings.NewReader(updateRoleBody))
	updateRoleReq.Header.Set("Authorization", "Bearer test-token")
	updateRoleReq.Header.Set("Content-Type", "application/json")
	updateRoleResp, err := client.Do(updateRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 200, updateRoleResp.StatusCode)
	updateRoleResp.Body.Close()

	// Verify rename was applied
	getRoleReq2, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles/my-role", nil)
	getRoleReq2.Header.Set("Authorization", "Bearer test-token")
	getRoleResp2, err := client.Do(getRoleReq2)
	require.NoError(t, err)
	assert.Equal(t, 200, getRoleResp2.StatusCode)
	var role2 postgres.Role
	require.NoError(t, json.NewDecoder(getRoleResp2.Body).Decode(&role2))
	require.NotNil(t, role2.Status)
	assert.Equal(t, "my_role_renamed", role2.Status.PostgresRole)
	getRoleResp2.Body.Close()

	// Delete role
	deleteRoleReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/2.0/postgres/projects/role-test-project/branches/main/roles/my-role", nil)
	deleteRoleReq.Header.Set("Authorization", "Bearer test-token")
	deleteRoleResp, err := client.Do(deleteRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteRoleResp.StatusCode)
	deleteRoleResp.Body.Close()
}

func TestPostgresRoleUpdateMaskPreservesUnmaskedFields(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	do := func(method, url, body string) *http.Response {
		req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		return resp
	}

	do(http.MethodPost, "/api/2.0/postgres/projects?project_id=mask-project", "").Body.Close()
	do(http.MethodPost, "/api/2.0/postgres/projects/mask-project/branches?branch_id=main", "").Body.Close()

	createBody := `{"spec":{"postgres_role":"app_role","attributes":{"createdb":false}}}`
	createResp := do(http.MethodPost, "/api/2.0/postgres/projects/mask-project/branches/main/roles?role_id=approle", createBody)
	require.Equal(t, 200, createResp.StatusCode)
	createResp.Body.Close()

	// Update only spec.attributes.createdb. The body also carries a different
	// postgres_role, which must be ignored because it is not named in update_mask.
	patchBody := `{"spec":{"postgres_role":"renamed","attributes":{"createdb":true}}}`
	patchResp := do(http.MethodPatch, "/api/2.0/postgres/projects/mask-project/branches/main/roles/approle?update_mask=spec.attributes.createdb", patchBody)
	assert.Equal(t, 200, patchResp.StatusCode)
	patchResp.Body.Close()

	getResp := do(http.MethodGet, "/api/2.0/postgres/projects/mask-project/branches/main/roles/approle", "")
	var role postgres.Role
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&role))
	getResp.Body.Close()

	require.NotNil(t, role.Status)
	require.NotNil(t, role.Status.Attributes)
	assert.True(t, role.Status.Attributes.Createdb, "masked field should be updated")
	assert.Equal(t, "app_role", role.Status.PostgresRole, "field absent from update_mask should be preserved")
}

func TestPostgresRoleNotFoundWhenBranchNotExists(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects?project_id=role-no-branch-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Try to create role without branch
	createRoleReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/postgres/projects/role-no-branch-project/branches/nonexistent/roles?role_id=my-role", nil)
	createRoleReq.Header.Set("Authorization", "Bearer test-token")
	createRoleResp, err := client.Do(createRoleReq)
	require.NoError(t, err)
	assert.Equal(t, 404, createRoleResp.StatusCode)
	createRoleResp.Body.Close()
}

func TestPostgresRoleCreateDuplicateReturns400(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	do := func(method, url, body string) *http.Response {
		req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		return resp
	}

	do(http.MethodPost, "/api/2.0/postgres/projects?project_id=dup-role-project", "").Body.Close()
	do(http.MethodPost, "/api/2.0/postgres/projects/dup-role-project/branches?branch_id=main", "").Body.Close()

	createBody := `{"spec":{"postgres_role":"app_role"}}`
	first := do(http.MethodPost, "/api/2.0/postgres/projects/dup-role-project/branches/main/roles?role_id=approle", createBody)
	require.Equal(t, 200, first.StatusCode)
	first.Body.Close()

	// Creating the same role again fails the way the real API does: 400, not 409.
	second := do(http.MethodPost, "/api/2.0/postgres/projects/dup-role-project/branches/main/roles?role_id=approle", createBody)
	assert.Equal(t, 400, second.StatusCode)
	second.Body.Close()
}
