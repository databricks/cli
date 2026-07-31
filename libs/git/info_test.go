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
	// Commit id written into the .git fixture; must be a full SHA-1 to parse.
	testDotGitCommit = "4fa481d6502eb6104d62c33a693ef8a134e765e0"
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

// writeDotGit lays down the subset of .git that fetchRepositoryInfoDotGit reads,
// matching what the new in-workspace Git folders expose on the FUSE mount.
func writeDotGit(t *testing.T, root, originURL, branch, commit string) {
	t.Helper()
	gitDir := filepath.Join(root, GitDirectoryName)
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0o755))

	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(gitDir, name), []byte(content), 0o600))
	}
	write("HEAD", "ref: refs/heads/"+branch+"\n")
	write("config", "[core]\n\trepositoryformatversion = 1\n[remote \"origin\"]\n\turl = "+originURL+"\n")
	write(filepath.Join("refs", "heads", branch), commit+"\n")
}

// On DBR, a bundle root with a readable .git is served from disk: the new type of
// in-workspace Git folder exposes one on the /Workspace FUSE mount, and get-status
// returns only id+path for it, so the API cannot supply the provenance anyway.
func TestFetchRepositoryInfoDbrPrefersDotGit(t *testing.T) {
	// The bundle root is a subdirectory, so the .git lookup has to walk up to find it.
	gitFolder := filepath.Join(t.TempDir(), "bundle-examples")
	bundleRoot := filepath.Join(gitFolder, "dabs_in_ws_bundle")
	require.NoError(t, os.MkdirAll(bundleRoot, 0o755))
	writeDotGit(t, gitFolder, testOriginURL, "main", testDotGitCommit)

	server := testserver.New(t)
	var apiCalled atomic.Bool
	server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
		apiCalled.Store(true)
		return testserver.Response{Body: map[string]any{}}
	})

	info, err := FetchRepositoryInfo(runtimeContext(t), bundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.False(t, apiCalled.Load(), "get-status should not be called when .git is readable")
	assert.Equal(t, testOriginURL, info.OriginURL)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, testDotGitCommit, info.LatestCommit)
	assert.Equal(t, gitFolder, info.WorktreeRoot)
}

// Without a .git on the mount, the API path still owns classic Repos and the older
// workspace Git folders, which return their git info inline.
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
			server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
				body := map[string]any{}
				if tt.gitInfo != nil {
					body["git_info"] = tt.gitInfo
				}
				return testserver.Response{Body: body}
			})

			info, err := FetchRepositoryInfo(runtimeContext(t), testBundleRoot, newTestWorkspaceClient(t, server))
			require.NoError(t, err)
			assert.Equal(t, tt.wantBranch, info.CurrentBranch)
			assert.Equal(t, tt.wantCommit, info.LatestCommit)
			assert.Equal(t, tt.wantOriginURL, info.OriginURL)
			assert.Equal(t, tt.wantWorktreeRoot, info.WorktreeRoot)
		})
	}
}

func TestHasDotGit(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "repo", "nested", "bundle")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "repo", GitDirectoryName), 0o755))

	assert.True(t, hasDotGit(filepath.Join(root, "repo")))
	assert.True(t, hasDotGit(nested), "should walk up to find .git")
	assert.False(t, hasDotGit(filepath.Join(t.TempDir(), "no-git-here")))
}
