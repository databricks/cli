package testserver_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createPostgresDatabaseTestBranch(t *testing.T, baseURL, projectID string) {
	t.Helper()
	do := func(method, path string) {
		t.Helper()
		req, err := http.NewRequest(method, baseURL+path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
	do(http.MethodPost, "/api/2.0/postgres/projects?project_id="+projectID)
	do(http.MethodPost, "/api/2.0/postgres/projects/"+projectID+"/branches?branch_id=main")
}

func TestPostgresDatabaseCreateRejectsMissingRole(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	createPostgresDatabaseTestBranch(t, server.URL, "missing-role")

	body := `{"spec":{"postgres_database":"app_db","role":"projects/missing-role/branches/main/roles/missing"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/2.0/postgres/projects/missing-role/branches/main/databases?database_id=appdb", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPostgresDatabaseCreateGeneratesValidOmittedID(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	createPostgresDatabaseTestBranch(t, server.URL, "generated-db")

	body := `{"spec":{"postgres_database":"app_db","role":"projects/generated-db/branches/main/roles/owner"}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/2.0/postgres/projects/generated-db/branches/main/databases", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getReq, err := http.NewRequest(http.MethodGet, server.URL+"/api/2.0/postgres/projects/generated-db/branches/main/databases", nil)
	require.NoError(t, err)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var databases struct {
		Databases []struct {
			Name string `json:"name"`
		} `json:"databases"`
	}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&databases))
	require.Len(t, databases.Databases, 1)
	assert.Regexp(t, `^projects/generated-db/branches/main/databases/db-[a-f0-9]{8}$`, databases.Databases[0].Name)
}

func TestPostgresProjectPermissionsRequireParent(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/2.0/permissions/database-projects/missing", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	req, err = http.NewRequest(http.MethodPost, server.URL+"/api/2.0/postgres/projects?project_id=present", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodGet, server.URL+"/api/2.0/permissions/database-projects/present", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodPut, server.URL+"/api/2.0/permissions/database-projects/present", strings.NewReader(`{"access_control_list":[]}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodPut, server.URL+"/api/2.0/permissions/database-projects/missing", strings.NewReader(`{"access_control_list":[]}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPostgresFakeRemovalDefaults(t *testing.T) {
	server := testserver.New(t)
	var projectPatch, endpointPatch map[string]any
	var callbackErr error
	server.RequestCallback = func(req *testserver.Request) {
		if req.Method != http.MethodPatch {
			return
		}
		target := &projectPatch
		if strings.Contains(req.URL.Path, "/endpoints/") {
			target = &endpointPatch
		}
		callbackErr = json.Unmarshal(req.Body, target)
	}
	testserver.AddDefaultHandlers(server)

	projectID := "removal-defaults"
	createPostgresDatabaseTestBranch(t, server.URL, projectID)
	projectPatchReq, err := http.NewRequest(http.MethodPatch, server.URL+"/api/2.0/postgres/projects/"+projectID+"?update_mask=spec.history_retention_duration", strings.NewReader(`{"spec":{"history_retention_duration":"604800s"}}`))
	require.NoError(t, err)
	projectPatchReq.Header.Set("Authorization", "Bearer test-token")
	projectResp, err := http.DefaultClient.Do(projectPatchReq)
	require.NoError(t, err)
	projectResp.Body.Close()
	require.NoError(t, callbackErr)
	require.Equal(t, http.StatusOK, projectResp.StatusCode)
	assert.Equal(t, "604800s", projectPatch["spec"].(map[string]any)["history_retention_duration"])

	endpointPatchReq, err := http.NewRequest(http.MethodPatch, server.URL+"/api/2.0/postgres/projects/"+projectID+"/branches/main/endpoints/primary?update_mask=spec.suspension", strings.NewReader(`{"spec":{"suspend_timeout_duration":"86400s"}}`))
	require.NoError(t, err)
	endpointPatchReq.Header.Set("Authorization", "Bearer test-token")
	endpointResp, err := http.DefaultClient.Do(endpointPatchReq)
	require.NoError(t, err)
	endpointResp.Body.Close()
	require.Equal(t, http.StatusOK, endpointResp.StatusCode)
	require.NoError(t, callbackErr)
	assert.Equal(t, "86400s", endpointPatch["spec"].(map[string]any)["suspend_timeout_duration"])
}
