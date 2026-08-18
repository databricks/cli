package git

import (
	"io/fs"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The cases that resolve metadata successfully are covered end to end by
// acceptance/bundle/debug/fetch-repository-info, which runs against both a fake
// and a real workspace. What is left here is what that test cannot reach.

func TestEnsureWorkspacePrefix(t *testing.T) {
	// get-status reports git_info.path without the /Workspace mount prefix, so it
	// is re-added to make the worktree root an absolute workspace path.
	assert.Equal(t, "/Workspace/Repos/me/repo", ensureWorkspacePrefix("/Repos/me/repo"))
	assert.Equal(t, "/Workspace/Repos/me/repo", ensureWorkspacePrefix("/Workspace/Repos/me/repo"))
}

// A path that does not exist is reported as fs.ErrNotExist, which
// FetchRepositoryInfo normalizes to no repository rather than an error.
func TestFetchRepositoryInfoAPI_MissingPath(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	const missing = "/Workspace/Users/me@example.com/nope"

	_, err = FetchRepositoryInfoAPI(t.Context(), missing, w)
	assert.ErrorIs(t, err, fs.ErrNotExist)

	ctx := dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: true, Version: "15.4"})
	info, err := FetchRepositoryInfo(ctx, missing, w)
	require.NoError(t, err)
	assert.Empty(t, info.WorktreeRoot)
}

// When the Repos API cannot be reached the worktree root is still reported, and
// the metadata is left empty rather than failing the call.
func TestFetchRepositoryInfoAPI_ReposGetFailure(t *testing.T) {
	const folderPath = "/Workspace/Users/me@example.com/gitfolder"

	server := testserver.New(t)
	// Registered before the default handlers: the first registration for a route
	// wins, so this replaces the Repos API read.
	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusForbidden,
			Body:       map[string]string{"message": "no access"},
		}
	})
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	_, err = w.Repos.Create(t.Context(), workspace.CreateRepoRequest{
		Url:      "https://github.com/databricks/cli",
		Provider: "gitHub",
		Path:     folderPath,
	})
	require.NoError(t, err)

	info, err := FetchRepositoryInfoAPI(t.Context(), folderPath, w)
	require.NoError(t, err)

	assert.Equal(t, folderPath, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
}
