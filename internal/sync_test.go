package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/bricks/cmd/sync"
	"github.com/databricks/bricks/libs/testfile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: these tests are bloated, refactor these, and make write down tests for
// all edge cases with interop between files, directory and notebooks during syncing
// https://databricks.atlassian.net/browse/DECO-416

// TODO: test case for mutiple files maybe
// TODO: Implement cache invalidation with full sync

// This test needs auth env vars to run.
// Please run using the deco env test or deco env shell
func setupRepo(t *testing.T, wsc *databricks.WorkspaceClient, ctx context.Context) (localRoot, remoteRoot string) {
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)
	repoUrl := "https://github.com/shreyas-goenka/empty-repo.git"
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-sync-integration-"))

	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     repoPath,
		Url:      repoUrl,
		Provider: "gitHub",
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, repoInfo.Id)
		assert.NoError(t, err)
	})

	// clone public empty remote repo
	tempDir := t.TempDir()
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = tempDir
	err = cmd.Run()
	assert.NoError(t, err)

	localRoot = filepath.Join(tempDir, "empty-repo")
	remoteRoot = repoPath
	return localRoot, remoteRoot
}

type assertSync struct {
	t          *testing.T
	c          *cobraTestRunner
	w          *databricks.WorkspaceClient
	localRoot  string
	remoteRoot string
}

func (a *assertSync) remoteDirContent(ctx context.Context, relativeDir string, expectedFiles []string) {
	remoteDir := path.Join(a.remoteRoot, relativeDir)
	a.c.Eventually(func() bool {
		objects, err := a.w.Workspace.ListAll(ctx, workspace.List{
			Path: remoteDir,
		})
		require.NoError(a.t, err)
		return len(objects) == len(expectedFiles)
	}, 30*time.Second, 5*time.Second)
	objects, err := a.w.Workspace.ListAll(ctx, workspace.List{
		Path: remoteDir,
	})
	require.NoError(a.t, err)

	var actualFiles []string
	for _, v := range objects {
		actualFiles = append(actualFiles, v.Path)
	}

	assert.Len(a.t, actualFiles, len(expectedFiles))
	for _, v := range expectedFiles {
		assert.Contains(a.t, actualFiles, path.Join(a.remoteRoot, relativeDir, v))
	}
}

// content should be a valid json object due to current go sdk limitations (31 December 2022, see https://github.com/databricks/databricks-sdk-go/pull/247)
func (a *assertSync) remoteFileContent(ctx context.Context, relativePath string, expectedContent string) {
	filePath := path.Join(a.remoteRoot, relativePath)

	// Remove leading "/" so we can use it in the URL.
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(filePath, "/"),
	)

	apiClient, err := client.New(a.w.Config)
	require.NoError(a.t, err)

	// Update to []byte after https://github.com/databricks/databricks-sdk-go/pull/247 is merged.
	var res json.RawMessage
	a.c.Eventually(func() bool {
		err = apiClient.Do(ctx, http.MethodGet, urlPath, nil, &res)
		require.NoError(a.t, err)
		actualContent := string(res)
		return actualContent == expectedContent
	}, 30*time.Second, 5*time.Second)
}

func (a *assertSync) objectType(ctx context.Context, relativePath string, expected string) {
	path := path.Join(a.remoteRoot, relativePath)

	a.c.Eventually(func() bool {
		metadata, err := a.w.Workspace.GetStatusByPath(ctx, path)
		require.NoError(a.t, err)
		return metadata.ObjectType.String() == expected
	}, 30*time.Second, 5*time.Second)
}

func (a *assertSync) language(ctx context.Context, relativePath string, expected string) {
	path := path.Join(a.remoteRoot, relativePath)

	a.c.Eventually(func() bool {
		metadata, err := a.w.Workspace.GetStatusByPath(ctx, path)
		require.NoError(a.t, err)
		return metadata.Language.String() == expected
	}, 30*time.Second, 5*time.Second)
}

func (a *assertSync) snapshotContains(files []string) {
	snapshotPath := filepath.Join(a.localRoot, ".databricks/sync-snapshots", sync.GetFileName(a.w.Config.Host, a.remoteRoot))
	assert.FileExists(a.t, snapshotPath)

	var s *sync.Snapshot
	f, err := os.Open(snapshotPath)
	assert.NoError(a.t, err)
	defer f.Close()

	bytes, err := io.ReadAll(f)
	assert.NoError(a.t, err)
	err = json.Unmarshal(bytes, &s)
	assert.NoError(a.t, err)

	assert.Equal(a.t, s.Host, a.w.Config.Host)
	assert.Equal(a.t, s.RemotePath, a.remoteRoot)
	for _, filePath := range files {
		_, ok := s.LastUpdatedTimes[filePath]
		assert.True(a.t, ok, fmt.Sprintf("%s not in snapshot file: %v", filePath, s.LastUpdatedTimes))
	}
	assert.Equal(a.t, len(files), len(s.LastUpdatedTimes))
}

func TestAccFullFileSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", localRepoPath)
	c := NewCobraTestRunner(t, "sync", "--remote-path", remoteRepoPath, "--persist-snapshot=false")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitkeep comes from cloning during repo setup
	assertSync.remoteDirContent(ctx, "", []string{".gitkeep"})

	// New file
	localFilePath := filepath.Join(localRepoPath, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", []string{"foo.txt", ".gitkeep", ".gitignore"})
	assertSync.remoteFileContent(ctx, "foo.txt", "")

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", []string{".gitkeep", ".gitignore"})
}

func TestAccIncrementalFileSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", localRepoPath)
	c := NewCobraTestRunner(t, "sync", "--remote-path", remoteRepoPath, "--persist-snapshot=true")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitkeep comes from cloning during repo setup
	assertSync.remoteDirContent(ctx, "", []string{".gitkeep"})

	// New file
	localFilePath := filepath.Join(localRepoPath, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", []string{"foo.txt", ".gitkeep", ".gitignore"})
	assertSync.remoteFileContent(ctx, "foo.txt", "")
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore", "foo.txt"})

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// delete
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", []string{".gitkeep", ".gitignore"})
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore"})
}

func TestAccIncrementalFolderSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", localRepoPath)
	c := NewCobraTestRunner(t, "sync", "--remote-path", remoteRepoPath, "--persist-snapshot=true")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitkeep comes from cloning during repo setup
	assertSync.remoteDirContent(ctx, "/", []string{".gitkeep"})

	// New file
	localFilePath := filepath.Join(localRepoPath, "dir1/dir2/dir3/foo.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", []string{"dir1", ".gitkeep", ".gitignore"})
	assertSync.remoteDirContent(ctx, "dir1", []string{"dir2"})
	assertSync.remoteDirContent(ctx, "dir1/dir2", []string{"dir3"})
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{"foo.txt"})
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore", "dir1/dir2/dir3/foo.txt"})

	// delete
	f.Remove(t)
	// directories are not cleaned up right now. This is not ideal
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{})
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore"})
}

// sync does not clean up empty directories from the workspace file system.
// This is a check for the edge case when a user does the following:
//
// 1. Add file foo/bar.txt
// 2. Delete foo/bar.txt (including the directory)
// 3. Add file foo
//
// In the above scenario sync should delete the empty folder and add foo to the remote
// file system
func TestAccIncrementalFileOverwritesFolder(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", localRepoPath)
	c := NewCobraTestRunner(t, "sync", "--remote-path", remoteRepoPath, "--persist-snapshot=true")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// create foo/bar.txt
	localFilePath := filepath.Join(localRepoPath, "foo/bar.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", []string{"foo", ".gitkeep", ".gitignore"})
	assertSync.remoteDirContent(ctx, "foo", []string{"bar.txt"})
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore", "foo/bar.txt"})

	// delete foo/bar.txt
	f.Remove(t)
	os.Remove(filepath.Join(localRepoPath, "foo"))
	assertSync.remoteDirContent(ctx, "foo", []string{})
	assertSync.objectType(ctx, "foo", "DIRECTORY")
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore"})

	f2 := testfile.CreateFile(t, filepath.Join(localRepoPath, "foo"))
	defer f2.Close(t)
	assertSync.remoteDirContent(ctx, "", []string{"foo", ".gitkeep", ".gitignore"})
	assertSync.objectType(ctx, "foo", "FILE")
	assertSync.snapshotContains([]string{".gitkeep", ".gitignore", "foo"})
}