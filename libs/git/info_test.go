package git

import (
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeWorkspace returns a client for a fake workspace with a Git folder
// created at repoPath. A Git folder under /Repos reports its metadata on
// get-status, while one outside /Repos has Git CLI access and does not, which is
// what makes the two paths through fetchRepositoryInfoAPI observable.
func newFakeWorkspace(t *testing.T, repoPath string) *databricks.WorkspaceClient {
	t.Helper()

	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	_, err = client.Repos.Create(t.Context(), workspace.CreateRepoRequest{
		Url:      "https://github.com/databricks/cli",
		Provider: "gitHub",
		Path:     repoPath,
	})
	require.NoError(t, err)

	return client
}

func TestFetchRepositoryInfoAPI_Repo(t *testing.T) {
	const repoPath = "/Workspace/Repos/me@example.com/repo"
	w := newFakeWorkspace(t, repoPath)

	info, err := fetchRepositoryInfoAPI(t.Context(), repoPath, w)
	require.NoError(t, err)

	assert.Equal(t, repoPath, info.WorktreeRoot)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.NotEmpty(t, info.LatestCommit)
	assert.Equal(t, "https://github.com/databricks/cli", info.OriginURL)
}

// A Git folder with Git CLI access reports no metadata on get-status, so it is
// read from the Repos API instead.
func TestFetchRepositoryInfoAPI_GitCliFolder(t *testing.T) {
	const folderPath = "/Workspace/Users/me@example.com/gitfolder"
	w := newFakeWorkspace(t, folderPath)

	info, err := fetchRepositoryInfoAPI(t.Context(), folderPath, w)
	require.NoError(t, err)

	assert.Equal(t, folderPath, info.WorktreeRoot)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.NotEmpty(t, info.LatestCommit)
	assert.Equal(t, "https://github.com/databricks/cli", info.OriginURL)
}

// A path inside a Git folder resolves the metadata of the folder root, which is
// also reported as the worktree root.
func TestFetchRepositoryInfoAPI_Subdirectory(t *testing.T) {
	const folderPath = "/Workspace/Users/me@example.com/gitfolder"
	w := newFakeWorkspace(t, folderPath)
	require.NoError(t, w.Workspace.MkdirsByPath(t.Context(), folderPath+"/a/b"))

	info, err := fetchRepositoryInfoAPI(t.Context(), folderPath+"/a/b", w)
	require.NoError(t, err)

	assert.Equal(t, folderPath, info.WorktreeRoot)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "https://github.com/databricks/cli", info.OriginURL)
}

// A path outside any Git folder has no metadata and is not an error.
func TestFetchRepositoryInfoAPI_NotAGitFolder(t *testing.T) {
	w := newFakeWorkspace(t, "/Workspace/Repos/me@example.com/repo")
	dir := "/Workspace/Users/me@example.com/plain"
	require.NoError(t, w.Workspace.MkdirsByPath(t.Context(), dir))

	info, err := FetchRepositoryInfoWorkspace(t.Context(), dir, w)
	require.NoError(t, err)

	assert.Empty(t, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
}

// A path that does not exist is reported as no repository rather than an error.
func TestFetchRepositoryInfoAPI_MissingPath(t *testing.T) {
	w := newFakeWorkspace(t, "/Workspace/Repos/me@example.com/repo")

	info, err := FetchRepositoryInfoWorkspace(t.Context(), "/Workspace/Users/me@example.com/nope", w)
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

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	_, err = client.Repos.Create(t.Context(), workspace.CreateRepoRequest{
		Url:      "https://github.com/databricks/cli",
		Provider: "gitHub",
		Path:     folderPath,
	})
	require.NoError(t, err)

	info, err := fetchRepositoryInfoAPI(t.Context(), folderPath, client)
	require.NoError(t, err)

	assert.Equal(t, folderPath, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
}

// The workspace API strips the /Workspace prefix from git_info.path, so the
// worktree root is reported with it re-added.
func TestFetchRepositoryInfoAPI_WorktreeRootKeepsWorkspacePrefix(t *testing.T) {
	w := newFakeWorkspace(t, "/Repos/me@example.com/repo")

	info, err := fetchRepositoryInfoAPI(t.Context(), "/Repos/me@example.com/repo", w)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(info.WorktreeRoot, "/Workspace/"), "got %q", info.WorktreeRoot)
}
