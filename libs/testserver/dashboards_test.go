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

// Cloud rejects a parent_path outside a real workspace folder. The fake mirrors
// that so a fuzzer value like "/etc/passwd" is a 400 rather than a stored path
// that later shows up as spurious drift. Existence alone is not enough: even an
// existing "/etc/passwd" directory is rejected because it isn't a workspace root.
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
