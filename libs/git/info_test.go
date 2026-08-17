package git

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRepositoryInfoAPIReadsMetadataFromReposAPI(t *testing.T) {
	// git-in-data-plane folders: get-status returns only id+path in git_info, so
	// branch/commit/url must be recovered from the Repos API by id.
	var statusPath string
	reposCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/workspace/get-status":
			statusPath = r.URL.Query().Get("path")
			_, _ = w.Write([]byte(`{"object_type":"DIRECTORY","git_info":{"id":123,"path":"/Repos/alice/proj"}}`))
		case "/api/2.0/repos/123":
			reposCalled = true
			_, _ = w.Write([]byte(`{"id":123,"branch":"main","head_commit_id":"abc123","url":"https://github.com/databricks/bundle-examples"}`))
		default:
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()

	w := databricks.Must(databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "dummy"}))

	info, err := fetchRepositoryInfoAPI(t.Context(), "/Workspace/Repos/alice/proj/sub/dir", w)
	require.NoError(t, err)

	assert.Equal(t, "/Workspace/Repos/alice/proj/sub/dir", statusPath)
	assert.True(t, reposCalled, "Repos API should be called to load git metadata")
	assert.Equal(t, "/Workspace/Repos/alice/proj", info.WorktreeRoot)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "abc123", info.LatestCommit)
	assert.Equal(t, "https://github.com/databricks/bundle-examples", info.OriginURL)
}

func TestFetchRepositoryInfoAPINotAGitFolder(t *testing.T) {
	// Outside any git folder get-status returns no git_info, so the result is
	// empty and the Repos API is never called.
	reposCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/2.0/repos/") {
			reposCalled = true
		}
		if r.URL.Path == "/api/2.0/workspace/get-status" {
			_, _ = w.Write([]byte(`{"object_type":"DIRECTORY"}`))
			return
		}
		// Ignore the SDK's host-metadata probe and anything else.
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	w := databricks.Must(databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "dummy"}))

	info, err := fetchRepositoryInfoAPI(t.Context(), "/Workspace/Users/alice/notebook", w)
	require.NoError(t, err)

	assert.False(t, reposCalled, "Repos API should not be called when the path is not a git folder")
	assert.Equal(t, RepositoryInfo{}, info)
}
