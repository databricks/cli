package testserver_test

import (
	"encoding/json"
	"net/http"
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
	createReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=test-project", nil)
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
	getReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/test-project", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getResp.StatusCode)

	var project postgres.Project
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&project))
	assert.Equal(t, "projects/test-project", project.Name)
	getResp.Body.Close()

	// List projects
	listReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects", nil)
	listReq.Header.Set("Authorization", "Bearer test-token")
	listResp, err := client.Do(listReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listResp.StatusCode)

	var listProjects postgres.ListProjectsResponse
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listProjects))
	assert.Len(t, listProjects.Projects, 1)
	listResp.Body.Close()

	// Delete project
	deleteReq, _ := http.NewRequest("DELETE", baseURL+"/api/2.0/postgres/projects/test-project", nil)
	deleteReq.Header.Set("Authorization", "Bearer test-token")
	deleteResp, err := client.Do(deleteReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteResp.StatusCode)
	deleteResp.Body.Close()

	// Verify project is deleted
	getReq2, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/test-project", nil)
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

	getReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/nonexistent", nil)
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
	createReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=dup-project", nil)
	createReq.Header.Set("Authorization", "Bearer test-token")
	createResp, err := client.Do(createReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createResp.StatusCode)
	createResp.Body.Close()

	// Try to create duplicate
	createReq2, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=dup-project", nil)
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
	createProjReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=branch-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects/branch-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Get branch
	getBranchReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/branch-test-project/branches/main", nil)
	getBranchReq.Header.Set("Authorization", "Bearer test-token")
	getBranchResp, err := client.Do(getBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getBranchResp.StatusCode)

	var branch postgres.Branch
	require.NoError(t, json.NewDecoder(getBranchResp.Body).Decode(&branch))
	assert.Equal(t, "projects/branch-test-project/branches/main", branch.Name)
	getBranchResp.Body.Close()

	// List branches
	listBranchReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/branch-test-project/branches", nil)
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
	deleteBranchReq, _ := http.NewRequest("DELETE", baseURL+"/api/2.0/postgres/projects/branch-test-project/branches/main", nil)
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
	createBranchReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects/nonexistent/branches?branch_id=main", nil)
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
	createProjReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=endpoint-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Create branch
	createBranchReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches?branch_id=main", nil)
	createBranchReq.Header.Set("Authorization", "Bearer test-token")
	createBranchResp, err := client.Do(createBranchReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createBranchResp.StatusCode)
	createBranchResp.Body.Close()

	// Create endpoint
	createEpReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints?endpoint_id=rw-endpoint", nil)
	createEpReq.Header.Set("Authorization", "Bearer test-token")
	createEpResp, err := client.Do(createEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createEpResp.StatusCode)
	createEpResp.Body.Close()

	// Get endpoint
	getEpReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", nil)
	getEpReq.Header.Set("Authorization", "Bearer test-token")
	getEpResp, err := client.Do(getEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, getEpResp.StatusCode)

	var endpoint postgres.Endpoint
	require.NoError(t, json.NewDecoder(getEpResp.Body).Decode(&endpoint))
	assert.Equal(t, "projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", endpoint.Name)
	getEpResp.Body.Close()

	// List endpoints
	listEpReq, _ := http.NewRequest("GET", baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints", nil)
	listEpReq.Header.Set("Authorization", "Bearer test-token")
	listEpResp, err := client.Do(listEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, listEpResp.StatusCode)

	var listEndpoints postgres.ListEndpointsResponse
	require.NoError(t, json.NewDecoder(listEpResp.Body).Decode(&listEndpoints))
	assert.Len(t, listEndpoints.Endpoints, 1)
	listEpResp.Body.Close()

	// Delete endpoint
	deleteEpReq, _ := http.NewRequest("DELETE", baseURL+"/api/2.0/postgres/projects/endpoint-test-project/branches/main/endpoints/rw-endpoint", nil)
	deleteEpReq.Header.Set("Authorization", "Bearer test-token")
	deleteEpResp, err := client.Do(deleteEpReq)
	require.NoError(t, err)
	assert.Equal(t, 200, deleteEpResp.StatusCode)
	deleteEpResp.Body.Close()
}

func TestPostgresEndpointNotFoundWhenBranchNotExists(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client := &http.Client{}
	baseURL := server.URL

	// Create project first
	createProjReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects?project_id=ep-test-project", nil)
	createProjReq.Header.Set("Authorization", "Bearer test-token")
	createProjResp, err := client.Do(createProjReq)
	require.NoError(t, err)
	assert.Equal(t, 200, createProjResp.StatusCode)
	createProjResp.Body.Close()

	// Try to create endpoint without branch
	createEpReq, _ := http.NewRequest("POST", baseURL+"/api/2.0/postgres/projects/ep-test-project/branches/nonexistent/endpoints?endpoint_id=rw-endpoint", nil)
	createEpReq.Header.Set("Authorization", "Bearer test-token")
	createEpResp, err := client.Do(createEpReq)
	require.NoError(t, err)
	assert.Equal(t, 404, createEpResp.StatusCode)
	createEpResp.Body.Close()
}
