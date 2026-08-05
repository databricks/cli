package git

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

// mockWorkspaceMount relocates the workspace mount point into a temporary directory
// and returns it. Without this the paths below do not start with the mount prefix, so
// FetchRepositoryInfo takes the on-disk branch on the prefix check alone and never
// reaches the .git lookup the tests are exercising.
func mockWorkspaceMount(t *testing.T) string {
	t.Helper()
	mount := filepath.Join(t.TempDir(), "Workspace")
	original := workspaceMountPrefix
	// Slash form, matching the real prefix and what the path helpers compare against.
	workspaceMountPrefix = filepath.ToSlash(mount) + "/"
	t.Cleanup(func() { workspaceMountPrefix = original })
	return mount
}

// mockGitFolder creates a Git folder at <mount>/Users/<user>/<name> holding a .git,
// plus the nested bundle root inside it, and returns both paths.
func mockGitFolder(t *testing.T, mount, name, branch, commit string) (gitFolder, bundleRoot string) {
	t.Helper()
	gitFolder = filepath.Join(mount, "Users", "test@databricks.com", name)
	bundleRoot = filepath.Join(gitFolder, "dabs_in_ws_bundle")
	require.NoError(t, os.MkdirAll(bundleRoot, 0o755))
	writeDotGit(t, gitFolder, testOriginURL, branch, commit)
	return gitFolder, bundleRoot
}

// writeDotGit lays down the subset of .git that fetchRepositoryInfoDotGit reads.
// A Git in Dataplane folder exposes a complete .git on the workspace mount (objects,
// index, packed-refs and all); libs/git only reads HEAD, config and the branch ref,
// so those three are all a fixture needs.
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

// newGetStatusServer returns a test server whose get-status records whether it was
// called and answers with the id+path-only git_info of a Git in Dataplane folder.
func newGetStatusServer(t *testing.T) (*testserver.Server, *atomic.Bool) {
	t.Helper()
	server := testserver.New(t)
	var called atomic.Bool
	server.Handle("GET", "/api/2.0/workspace/get-status", func(_ testserver.Request) any {
		called.Store(true)
		return testserver.Response{Body: map[string]any{
			"git_info": map[string]any{"id": testRepoID, "path": testGitFolderRaw},
		}}
	})
	return server, &called
}

// On DBR, a bundle root whose Git folder has a readable .git is served from disk:
// Git in Dataplane folders expose one on the workspace mount, and get-status returns
// only id+path for them, so the API cannot supply the provenance anyway.
func TestFetchRepositoryInfoDbrPrefersDotGit(t *testing.T) {
	mount := mockWorkspaceMount(t)
	// The bundle root is a subdirectory, so the .git lookup has to walk up to find it.
	gitFolder, bundleRoot := mockGitFolder(t, mount, "bundle-examples", "main", testDotGitCommit)

	server, apiCalled := newGetStatusServer(t)

	info, err := FetchRepositoryInfo(runtimeContext(t), bundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.False(t, apiCalled.Load(), "get-status should not be called when .git is readable")
	assert.Equal(t, testOriginURL, info.OriginURL)
	assert.Equal(t, "main", info.CurrentBranch)
	assert.Equal(t, testDotGitCommit, info.LatestCommit)
	assert.Equal(t, gitFolder, info.WorktreeRoot)
}

// The counterpart of the test above: on the same mount, a bundle root with no .git
// anywhere below the Git folder boundary must still reach the API. Without this the
// assertion above passes for any routing rule that happens to prefer disk.
func TestFetchRepositoryInfoDbrWithoutDotGitUsesAPI(t *testing.T) {
	mount := mockWorkspaceMount(t)
	bundleRoot := filepath.Join(mount, "Users", "test@databricks.com", "plain-dir", "dabs_in_ws_bundle")
	require.NoError(t, os.MkdirAll(bundleRoot, 0o755))

	server, apiCalled := newGetStatusServer(t)

	info, err := FetchRepositoryInfo(runtimeContext(t), bundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.True(t, apiCalled.Load(), "get-status must be called when no .git is reachable")
	assert.Equal(t, testWorktreeRoot, info.WorktreeRoot)
}

// A .git above the Git folder boundary belongs to a different repository, so it must
// not be reported as this bundle's provenance. Reproduced on DBR: a `git init` in the
// user's workspace home otherwise makes every bundle below it report that repo's
// origin, branch and commit, which can also fail the deploy in ValidateGitDetails.
func TestFetchRepositoryInfoDbrIgnoresDotGitAboveGitFolder(t *testing.T) {
	mount := mockWorkspaceMount(t)
	userHome := filepath.Join(mount, "Users", "test@databricks.com")
	bundleRoot := filepath.Join(userHome, "plain-dir", "dabs_in_ws_bundle")
	require.NoError(t, os.MkdirAll(bundleRoot, 0o755))
	// An unrelated repository in the user's workspace home, above the boundary.
	writeDotGit(t, userHome, "https://github.com/unrelated/other.git", "other-branch", testDotGitCommit)

	server, apiCalled := newGetStatusServer(t)

	info, err := FetchRepositoryInfo(runtimeContext(t), bundleRoot, newTestWorkspaceClient(t, server))
	require.NoError(t, err)
	assert.True(t, apiCalled.Load(), "get-status must be called instead of reading an unrelated .git")
	assert.Empty(t, info.OriginURL)
	assert.Empty(t, info.CurrentBranch)
	assert.Equal(t, testWorktreeRoot, info.WorktreeRoot)
}

// Without a .git on the mount, the API path still owns classic Repos, which return
// their git info inline.
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
			// A Git in Dataplane folder whose .git is not readable: get-status
			// carries only id+path, so there is no provenance to report.
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
	mount := mockWorkspaceMount(t)
	gitFolder, bundleRoot := mockGitFolder(t, mount, "bundle-examples", "main", testDotGitCommit)
	ctx := t.Context()

	assert.True(t, hasDotGit(ctx, gitFolder))
	assert.True(t, hasDotGit(ctx, bundleRoot), "should walk up to find .git")
	assert.False(t, hasDotGit(ctx, filepath.Join(mount, "Users", "test@databricks.com", "no-git-here")))
}

// Workspace paths are always slash-separated, but on Windows the bundle root reaches
// these helpers with backslashes, so every comparison must normalize first. Without it
// the mount prefix never matches and the walk escapes the owner root. Windows-only:
// filepath.ToSlash is a no-op elsewhere, so there is nothing to convert.
func TestPathHelpersNormalizeSeparators(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("path separator normalization only differs on Windows")
	}

	original := workspaceMountPrefix
	workspaceMountPrefix = "C:/tmp/Workspace/"
	t.Cleanup(func() { workspaceMountPrefix = original })

	backslashed := `C:\tmp\Workspace\Users\me@databricks.com\repo\bundle`
	assert.True(t, isWorkspaceMountPath(backslashed))
	assert.Equal(t, "C:/tmp/Workspace/Users/me@databricks.com", ownerRoot(backslashed))
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
