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

	_ "github.com/databricks/bricks/cmd/sync"
	"github.com/databricks/bricks/libs/sync"
	"github.com/databricks/bricks/libs/testfile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	repoUrl   = "https://github.com/databricks/databricks-empty-ide-project.git"
	repoFiles = []string{"README-IDE.md"}
)

// This test needs auth env vars to run.
// Please run using the deco env test or deco env shell
func setupRepo(t *testing.T, wsc *databricks.WorkspaceClient, ctx context.Context) (localRoot, remoteRoot string) {
	me, err := wsc.CurrentUser.Me(ctx)
	require.NoError(t, err)
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-sync-integration-"))

	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     repoPath,
		Url:      repoUrl,
		Provider: "gitHub",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, repoInfo.Id)
		assert.NoError(t, err)
	})

	tempDir := t.TempDir()
	localRoot = filepath.Join(tempDir, "empty-repo")
	remoteRoot = repoPath

	// clone public empty remote repo
	cmd := exec.Command("git", "clone", repoUrl, localRoot)
	err = cmd.Run()
	require.NoError(t, err)
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

func (a *assertSync) remoteFileContent(ctx context.Context, relativePath string, expectedContent string) {
	filePath := path.Join(a.remoteRoot, relativePath)

	// Remove leading "/" so we can use it in the URL.
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(filePath, "/"),
	)

	apiClient, err := client.New(a.w.Config)
	require.NoError(a.t, err)

	var res []byte
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
		if err != nil {
			return false
		}
		return metadata.ObjectType.String() == expected
	}, 30*time.Second, 5*time.Second)
}

func (a *assertSync) language(ctx context.Context, relativePath string, expected string) {
	path := path.Join(a.remoteRoot, relativePath)

	a.c.Eventually(func() bool {
		metadata, err := a.w.Workspace.GetStatusByPath(ctx, path)
		if err != nil {
			return false
		}
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
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--full", "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitignore is created by the sync process to enforce .databricks is not synced
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))

	// New file
	localFilePath := filepath.Join(localRepoPath, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.txt", ".gitignore"))
	assertSync.remoteFileContent(ctx, "foo.txt", "")

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))
}

func TestAccIncrementalFileSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitignore is created by the sync process to enforce .databricks is not synced
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))

	// New file
	localFilePath := filepath.Join(localRepoPath, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.txt", ".gitignore"))
	assertSync.remoteFileContent(ctx, "foo.txt", "")
	assertSync.snapshotContains(append(repoFiles, "foo.txt", ".gitignore"))

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))
	assertSync.snapshotContains(append(repoFiles, ".gitignore"))
}

func TestAccNestedFolderSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// .gitignore is created by the sync process to enforce .databricks is not synced
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))

	// New file
	localFilePath := filepath.Join(localRepoPath, "dir1/dir2/dir3/foo.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "dir1"))
	assertSync.remoteDirContent(ctx, "dir1", []string{"dir2"})
	assertSync.remoteDirContent(ctx, "dir1/dir2", []string{"dir3"})
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{"foo.txt"})
	assertSync.snapshotContains(append(repoFiles, ".gitignore", filepath.FromSlash("dir1/dir2/dir3/foo.txt")))

	// delete
	f.Remove(t)
	// directories are not cleaned up right now. This is not ideal
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{})
	assertSync.snapshotContains(append(repoFiles, ".gitignore"))
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
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
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
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))
	assertSync.remoteDirContent(ctx, "foo", []string{"bar.txt"})
	assertSync.snapshotContains(append(repoFiles, ".gitignore", filepath.FromSlash("foo/bar.txt")))

	// delete foo/bar.txt
	f.Remove(t)
	os.Remove(filepath.Join(localRepoPath, "foo"))
	assertSync.remoteDirContent(ctx, "foo", []string{})
	assertSync.objectType(ctx, "foo", "DIRECTORY")
	assertSync.snapshotContains(append(repoFiles, ".gitignore"))

	f2 := testfile.CreateFile(t, filepath.Join(localRepoPath, "foo"))
	defer f2.Close(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo"))
	assertSync.objectType(ctx, "foo", "FILE")
	assertSync.snapshotContains(append(repoFiles, ".gitignore", "foo"))
}

func TestAccIncrementalSyncPythonNotebookToFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// create python notebook
	localFilePath := filepath.Join(localRepoPath, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	f.Overwrite(t, "# Databricks notebook source")

	// Run `bricks sync` in the background.
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// notebook was uploaded properly
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo"))
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")
	assertSync.snapshotContains(append(repoFiles, ".gitignore", "foo.py"))

	// convert to vanilla python file
	f.Overwrite(t, "# No longer a python notebook")
	assertSync.objectType(ctx, "foo.py", "FILE")
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo.py"))
	assertSync.snapshotContains(append(repoFiles, ".gitignore", "foo.py"))

	// delete the vanilla python file
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))
	assertSync.snapshotContains(append(repoFiles, ".gitignore"))
}

func TestAccIncrementalSyncFileToPythonNotebook(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Run `bricks sync` in the background.
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// create vanilla python file
	localFilePath := filepath.Join(localRepoPath, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)

	// assert file upload
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo.py"))
	assertSync.objectType(ctx, "foo.py", "FILE")
	assertSync.snapshotContains(append(repoFiles, ".gitignore", "foo.py"))

	// convert to notebook
	f.Overwrite(t, "# Databricks notebook source")
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo"))
	assertSync.snapshotContains(append(repoFiles, ".gitignore", "foo.py"))
}

func TestAccIncrementalSyncPythonNotebookDelete(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
	ctx := context.Background()

	localRepoPath, remoteRepoPath := setupRepo(t, wsc, ctx)

	// create python notebook
	localFilePath := filepath.Join(localRepoPath, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	f.Overwrite(t, "# Databricks notebook source")

	// Run `bricks sync` in the background.
	c := NewCobraTestRunner(t, "sync", localRepoPath, remoteRepoPath, "--watch")
	c.RunBackground()

	assertSync := assertSync{
		t:          t,
		c:          c,
		w:          wsc,
		localRoot:  localRepoPath,
		remoteRoot: remoteRepoPath,
	}

	// notebook was uploaded properly
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore", "foo"))
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")

	// Delete notebook
	f.Remove(t)
	assertSync.remoteDirContent(ctx, "", append(repoFiles, ".gitignore"))
}
