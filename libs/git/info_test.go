package git

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
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

// A /Workspace path on DBR with no reachable .git goes to the API, which owns classic
// Repos (they return their git info inline) and is also all a Git in Dataplane folder
// has left when its .git cannot be read. The test host has no /Workspace, so hasDotGit
// finds nothing and every case here exercises the API branch.
func TestFetchRepositoryInfoAPI(t *testing.T) {
	tests := []struct {
		name string
		// gitInfo is the git_info object returned by get-status; nil means it is omitted.
		gitInfo map[string]any

		wantBranch       string
		wantCommit       string
		wantOriginURL    string
		wantWorktreeRoot string
	}{
		{
			name: "git folder returns full git info inline",
			gitInfo: map[string]any{
				"id":             testRepoID,
				"path":           testGitFolderRaw,
				"branch":         "main",
				"head_commit_id": "abc123",
				"url":            testOriginURL,
			},
			wantBranch:       "main",
			wantCommit:       "abc123",
			wantOriginURL:    testOriginURL,
			wantWorktreeRoot: testWorktreeRoot,
		},
		{
			// A remoteless folder has no origin URL to report.
			name: "remoteless git folder reports branch and commit but no url",
			gitInfo: map[string]any{
				"id":             testRepoID,
				"path":           testGitFolderRaw,
				"branch":         "main",
				"head_commit_id": "abc123",
			},
			wantBranch:       "main",
			wantCommit:       "abc123",
			wantOriginURL:    "",
			wantWorktreeRoot: testWorktreeRoot,
		},
		{
			// The Git in Dataplane response this change exists for: id+path only, so
			// there is no provenance to report and the .git on the mount is the only
			// source. Covered end to end by acceptance/bundle/git-info on DBR.
			name: "id-only git info yields no provenance",
			gitInfo: map[string]any{
				"id":   testRepoID,
				"path": testGitFolderRaw,
			},
			wantBranch:       "",
			wantCommit:       "",
			wantOriginURL:    "",
			wantWorktreeRoot: testWorktreeRoot,
		},
		{
			// A plain workspace directory: no git info, and no error either.
			name:             "no git info degrades to empty provenance",
			gitInfo:          nil,
			wantBranch:       "",
			wantCommit:       "",
			wantOriginURL:    "",
			wantWorktreeRoot: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			var apiCalled atomic.Bool
			server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
				apiCalled.Store(true)
				body := map[string]any{}
				if tt.gitInfo != nil {
					body["git_info"] = tt.gitInfo
				}
				return testserver.Response{Body: body}
			})

			info, err := FetchRepositoryInfo(runtimeContext(t), testBundleRoot, newTestWorkspaceClient(t, server))
			require.NoError(t, err)
			assert.True(t, apiCalled.Load(), "no .git is reachable, so get-status must be called")
			assert.Equal(t, tt.wantBranch, info.CurrentBranch)
			assert.Equal(t, tt.wantCommit, info.LatestCommit)
			assert.Equal(t, tt.wantOriginURL, info.OriginURL)
			assert.Equal(t, tt.wantWorktreeRoot, info.WorktreeRoot)
		})
	}
}

// findDotGitBelow is what decides whether provenance comes from disk, and its ceiling
// is what keeps a .git outside the bundle's own Git folder from being picked up.
func TestFindDotGitBelow(t *testing.T) {
	// Stands in for the owner root (/Workspace/Users/<user>): Git folders live inside
	// it, so a .git at or above it belongs to a different repository.
	ceiling := filepath.ToSlash(t.TempDir())
	gitFolder := filepath.Join(ceiling, "bundle-examples")
	bundleRoot := filepath.Join(gitFolder, "dabs_in_ws_bundle", "nested")
	require.NoError(t, os.MkdirAll(bundleRoot, 0o755))

	// No .git anywhere yet.
	root, err := findDotGitBelow(filepath.ToSlash(bundleRoot), ceiling)
	require.NoError(t, err)
	assert.Empty(t, root)

	// A .git at the ceiling itself is out of bounds: without the bound this would be
	// reported as the bundle's provenance, and could fail the deploy in
	// ValidateGitDetails by comparing against an unrelated repository's branch.
	require.NoError(t, os.MkdirAll(filepath.Join(ceiling, GitDirectoryName), 0o755))
	root, err = findDotGitBelow(filepath.ToSlash(bundleRoot), ceiling)
	require.NoError(t, err)
	assert.Empty(t, root, "a .git at the owner root must not be picked up")

	// The Git folder's own .git is found, including by walking up from a subdirectory.
	require.NoError(t, os.MkdirAll(filepath.Join(gitFolder, GitDirectoryName), 0o755))
	root, err = findDotGitBelow(filepath.ToSlash(bundleRoot), ceiling)
	require.NoError(t, err)
	assert.Equal(t, filepath.ToSlash(gitFolder), root)

	root, err = findDotGitBelow(filepath.ToSlash(gitFolder), ceiling)
	require.NoError(t, err)
	assert.Equal(t, filepath.ToSlash(gitFolder), root)
}

func TestOwnerRoot(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/Workspace/Users/me@databricks.com/repo/bundle", "/Workspace/Users/me@databricks.com"},
		{"/Workspace/Users/me@databricks.com/repo", "/Workspace/Users/me@databricks.com"},
		{"/Workspace/Users/me@databricks.com", "/Workspace/Users/me@databricks.com"},
		{"/Workspace/Repos/me@databricks.com/repo", "/Workspace/Repos/me@databricks.com"},
		{"/Workspace/Shared/repo/bundle", "/Workspace/Shared"},
		{"/Workspace/Users", "/Workspace/Users"},
		// Not under the mount at all: nothing above the mount is ever searched.
		{"/tmp/local/bundle", "/Workspace"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, ownerRoot(tt.path))
		})
	}
}

func TestHasDotGit(t *testing.T) {
	// A /Workspace path cannot exist on the test host, so nothing is reachable.
	assert.False(t, hasDotGit(t.Context(), testBundleRoot))
}
