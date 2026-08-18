package git

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepoPath = "/Workspace/Repos/me@example.com/repo"

// newTestWorkspaceClient serves the given get-status and repos/{id} response
// bodies and returns a client pointing at them, plus the paths requested.
func newTestWorkspaceClient(t *testing.T, getStatus, reposGet string) (*databricks.WorkspaceClient, *[]string) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK also probes /.well-known/databricks-config; only record API calls.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			requested = append(requested, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/2.0/workspace/get-status":
			_, err := w.Write([]byte(getStatus))
			assert.NoError(t, err)
		case "/api/2.0/repos/123":
			if reposGet == "" {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"nope"}`))
				assert.NoError(t, err)
				return
			}
			_, err := w.Write([]byte(reposGet))
			assert.NoError(t, err)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client, &requested
}

func TestFetchRepositoryInfoAPIFullGitInfo(t *testing.T) {
	getStatus, err := json.Marshal(map[string]any{
		"object_type": "REPO",
		"path":        testRepoPath,
		"git_info": map[string]any{
			"id":             123,
			"path":           "/Repos/me@example.com/repo",
			"branch":         "main",
			"head_commit_id": "abc123",
			"url":            "https://github.com/databricks/cli",
		},
	})
	require.NoError(t, err)

	w, requested := newTestWorkspaceClient(t, string(getStatus), "")
	info, err := fetchRepositoryInfoAPI(t.Context(), testRepoPath, w)
	require.NoError(t, err)

	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "abc123", info.LatestCommit)
	assert.Equal(t, "https://github.com/databricks/cli", info.OriginURL)
	assert.Equal(t, testRepoPath, info.WorktreeRoot)

	// The metadata was already present, so the Repos API is not consulted.
	assert.Equal(t, []string{"/api/2.0/workspace/get-status"}, *requested)
}

// A Git folder with Git CLI access reports only id and path, so the metadata is
// recovered from the Repos API.
func TestFetchRepositoryInfoAPIGitCliFolderFallsBackToReposGet(t *testing.T) {
	getStatus, err := json.Marshal(map[string]any{
		"object_type":    "DIRECTORY",
		"path":           testRepoPath,
		"directory_info": map[string]any{"is_git_folder": true},
		"git_info": map[string]any{
			"id":   123,
			"path": "/Repos/me@example.com/repo",
		},
	})
	require.NoError(t, err)

	reposGet, err := json.Marshal(map[string]any{
		"id":             123,
		"path":           "/Repos/me@example.com/repo",
		"branch":         "main",
		"head_commit_id": "abc123",
		"url":            "https://github.com/databricks/cli",
	})
	require.NoError(t, err)

	w, requested := newTestWorkspaceClient(t, string(getStatus), string(reposGet))
	info, err := fetchRepositoryInfoAPI(t.Context(), testRepoPath, w)
	require.NoError(t, err)

	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "abc123", info.LatestCommit)
	assert.Equal(t, "https://github.com/databricks/cli", info.OriginURL)
	assert.Equal(t, testRepoPath, info.WorktreeRoot)

	assert.Equal(t, []string{"/api/2.0/workspace/get-status", "/api/2.0/repos/123"}, *requested)
}

// git_info.id points at the Git folder root, so a subdirectory resolves the same
// metadata and keeps the root as the worktree root.
func TestFetchRepositoryInfoAPIGitCliFolderSubdir(t *testing.T) {
	getStatus, err := json.Marshal(map[string]any{
		"object_type":    "DIRECTORY",
		"path":           testRepoPath + "/subdir",
		"directory_info": map[string]any{"is_git_folder": false},
		"git_info": map[string]any{
			"id":   123,
			"path": "/Repos/me@example.com/repo",
		},
	})
	require.NoError(t, err)

	reposGet, err := json.Marshal(map[string]any{
		"id":             123,
		"branch":         "main",
		"head_commit_id": "abc123",
		"url":            "https://github.com/databricks/cli",
	})
	require.NoError(t, err)

	w, _ := newTestWorkspaceClient(t, string(getStatus), string(reposGet))
	info, err := fetchRepositoryInfoAPI(t.Context(), testRepoPath+"/subdir", w)
	require.NoError(t, err)

	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, testRepoPath, info.WorktreeRoot)
}

// A failing Repos lookup keeps the worktree root and leaves metadata empty.
func TestFetchRepositoryInfoAPIReposGetFailure(t *testing.T) {
	getStatus, err := json.Marshal(map[string]any{
		"object_type": "DIRECTORY",
		"path":        testRepoPath,
		"git_info": map[string]any{
			"id":   123,
			"path": "/Repos/me@example.com/repo",
		},
	})
	require.NoError(t, err)

	w, _ := newTestWorkspaceClient(t, string(getStatus), "")
	info, err := fetchRepositoryInfoAPI(t.Context(), testRepoPath, w)
	require.NoError(t, err)

	assert.Equal(t, testRepoPath, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
}

// A path outside a Git folder has no git_info and must not trigger a Repos lookup.
func TestFetchRepositoryInfoAPINoGitInfo(t *testing.T) {
	getStatus, err := json.Marshal(map[string]any{
		"object_type": "DIRECTORY",
		"path":        "/Workspace/Users/me@example.com/dir",
	})
	require.NoError(t, err)

	w, requested := newTestWorkspaceClient(t, string(getStatus), "")
	info, err := fetchRepositoryInfoAPI(t.Context(), "/Workspace/Users/me@example.com/dir", w)
	require.NoError(t, err)

	assert.Empty(t, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Equal(t, []string{"/api/2.0/workspace/get-status"}, *requested)
}
