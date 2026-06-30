package testserver_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func importFile(t *testing.T, baseURL, path, body string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/workspace-files/import-file/"+strings.TrimPrefix(path, "/")+"?overwrite=true", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func mkdirs(t *testing.T, baseURL, path string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/workspace/mkdirs", strings.NewReader(`{"path":"`+path+`"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
}

func getStatus(t *testing.T, baseURL, path string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/workspace/get-status?path="+path, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// The real import API returns a 404 when the parent directory does not exist; it
// does not create it. Callers get "mkdir -p" semantics only by calling mkdirs first.
func TestWorkspaceImportRejectsMissingParent(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	assert.Equal(t, 404, importFile(t, server.URL, "/test-dir/file.py", "content"))

	mkdirs(t, server.URL, "/test-dir")
	assert.Equal(t, 200, importFile(t, server.URL, "/test-dir/file.py", "content"))
}

// mkdirs creates all intermediate directories, matching "mkdir -p".
func TestWorkspaceMkdirsRecursive(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	mkdirs(t, server.URL, "/a/b/c")

	for _, dir := range []string{"/a", "/a/b", "/a/b/c"} {
		assert.Equal(t, 200, getStatus(t, server.URL, dir), dir)
	}
}
