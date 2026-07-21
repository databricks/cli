package testserver_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createDashboard(t *testing.T, baseURL, parentPath string) int {
	t.Helper()
	body := `{"display_name":"d","parent_path":"` + parentPath + `","warehouse_id":"w"}`
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/lakeview/dashboards", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// A non-workspace parent_path is a 400, even when the directory exists: "/etc/passwd"
// isn't under a workspace root, so it's rejected rather than stored as drift.
func TestDashboardCreateRejectsInvalidParentPath(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	mkdirs(t, server.URL, "/Users/dev")
	mkdirs(t, server.URL, "/Workspace/Users/dev")
	assert.Equal(t, 200, createDashboard(t, server.URL, "/Users/dev"))
	assert.Equal(t, 200, createDashboard(t, server.URL, "/Workspace/Users/dev"))

	mkdirs(t, server.URL, "/etc/passwd")
	assert.Equal(t, 400, createDashboard(t, server.URL, "/etc/passwd"))
	assert.Equal(t, 400, createDashboard(t, server.URL, "/Users/../etc"))
	assert.Equal(t, 400, createDashboard(t, server.URL, "relative/path"))
	assert.Equal(t, 400, createDashboard(t, server.URL, "/Users/\x01dev"))
}
