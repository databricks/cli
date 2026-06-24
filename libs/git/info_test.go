package git

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Bundle root passed to FetchRepositoryInfo: a subdirectory of the git folder.
	testBundleRoot = "/Workspace/Users/test/bundle-examples/dabs_in_ws_bundle"
	// Git folder path as get-status returns it (without the /Workspace prefix).
	testGitFolderRaw = "/Users/test/bundle-examples"
	// Expected worktree root after ensureWorkspacePrefix is applied.
	testWorktreeRoot = "/Workspace/Users/test/bundle-examples"
	testRepoID       = int64(2884540697170475)
	testOriginURL    = "https://github.com/databricks/bundle-examples.git"
)

func newTestWorkspaceClient(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return w
}

// runtimeContext forces the in-workspace API branch of FetchRepositoryInfo
// without needing a real /databricks directory on the test host.
func runtimeContext(t *testing.T) context.Context {
	return dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: true, Version: "15.4"})
}

// New workspace git folders return only id+path from get-status; the missing
// branch/commit/url are recovered from the Repos API by id.
func TestFetchRepositoryInfoNewGitFolderFallsBackToReposAPI(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
		return testserver.Response{Body: map[string]any{
			"git_info": map[string]any{
				"id":   testRepoID,
				"path": testGitFolderRaw,
			},
		}}
	})
	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(_ testserver.Request) any {
		return testserver.Response{Body: workspace.GetRepoResponse{
			Id:           testRepoID,
			Branch:       "main",
			HeadCommitId: "d53214abc",
			Url:          testOriginURL,
			Provider:     "gitHub",
			Path:         testGitFolderRaw,
		}}
	})

	info, err := FetchRepositoryInfo(runtimeContext(t), testBundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "d53214abc", info.LatestCommit)
	assert.Equal(t, testOriginURL, info.OriginURL)
	assert.Equal(t, testWorktreeRoot, info.WorktreeRoot)
}

// Classic Repos return full git info inline from get-status, so the Repos API is
// not called.
func TestFetchRepositoryInfoClassicRepoSkipsReposAPI(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
		return testserver.Response{Body: map[string]any{
			"git_info": map[string]any{
				"id":             testRepoID,
				"path":           testGitFolderRaw,
				"branch":         "main",
				"head_commit_id": "abc123",
				"url":            testOriginURL,
			},
		}}
	})
	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(_ testserver.Request) any {
		t.Error("Repos API must not be called when get-status returns the URL inline")
		return testserver.Response{StatusCode: 500}
	})

	info, err := FetchRepositoryInfo(runtimeContext(t), testBundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, "abc123", info.LatestCommit)
	assert.Equal(t, testOriginURL, info.OriginURL)
	assert.Equal(t, testWorktreeRoot, info.WorktreeRoot)
}

// A failed Repos lookup must not fail the deploy: the worktree root stays set and
// the provenance fields stay empty, with no error.
func TestFetchRepositoryInfoReposLookupFailureDegradesGracefully(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
		return testserver.Response{Body: map[string]any{
			"git_info": map[string]any{
				"id":   testRepoID,
				"path": testGitFolderRaw,
			},
		}}
	})
	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(_ testserver.Request) any {
		return testserver.Response{StatusCode: 404, Body: map[string]string{"message": "not found"}}
	})

	info, err := FetchRepositoryInfo(runtimeContext(t), testBundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
	assert.Equal(t, testWorktreeRoot, info.WorktreeRoot)
}
