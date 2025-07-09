package sync_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/testfile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	repoUrl   = "https://github.com/databricks/databricks-empty-ide-project.git"
	repoFiles = []string{}
)

// This test needs auth env vars to run.
// Please run using the deco env test or deco env shell
func setupRepo(t *testing.T, wsc *databricks.WorkspaceClient, ctx context.Context) (localRoot, remoteRoot string) {
	me, err := wsc.CurrentUser.Me(ctx)
	require.NoError(t, err)
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, testutil.RandomName("empty-repo-sync-integration-"))

	repoInfo, err := wsc.Repos.Create(ctx, workspace.CreateRepoRequest{
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

type syncTest struct {
	t          *testing.T
	c          *testcli.Runner
	w          *databricks.WorkspaceClient
	f          filer.Filer
	localRoot  string
	remoteRoot string
}

func setupSyncTest(t *testing.T, args ...string) (context.Context, *syncTest) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	localRoot := t.TempDir()
	remoteRoot := acc.TemporaryWorkspaceDir(wt, "sync-")
	f, err := filer.NewWorkspaceFilesClient(w, remoteRoot)
	require.NoError(t, err)

	// Prepend common arguments.
	args = append([]string{
		"sync",
		localRoot,
		remoteRoot,
		"--output",
		"json",
	}, args...)

	c := testcli.NewRunner(t, ctx, args...)
	c.RunBackground()

	return ctx, &syncTest{
		t:          t,
		c:          c,
		w:          w,
		f:          f,
		localRoot:  localRoot,
		remoteRoot: remoteRoot,
	}
}

func (s *syncTest) waitForCompletionMarker() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			s.t.Fatal("timed out waiting for sync to complete")
		case line := <-s.c.StdoutLines:
			var event sync.EventBase
			err := json.Unmarshal([]byte(line), &event)
			require.NoError(s.t, err)
			if event.Type == sync.EventTypeComplete {
				return
			}
		}
	}
}

func (a *syncTest) remoteDirContent(ctx context.Context, relativeDir string, expectedFiles []string) {
	remoteDir := path.Join(a.remoteRoot, relativeDir)
	a.c.Eventually(func() bool {
		objects, err := a.w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
			Path: remoteDir,
		})
		require.NoError(a.t, err)
		return len(objects) == len(expectedFiles)
	}, 30*time.Second, 5*time.Second)
	objects, err := a.w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
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

func (a *syncTest) remoteFileContent(ctx context.Context, relativePath, expectedContent string) {
	filePath := path.Join(a.remoteRoot, relativePath)

	// Remove leading "/" so we can use it in the URL.
	urlPath := "/api/2.0/workspace-files/" + strings.TrimLeft(filePath, "/")

	apiClient, err := client.New(a.w.Config)
	require.NoError(a.t, err)

	var res []byte
	a.c.Eventually(func() bool {
		err = apiClient.Do(ctx, http.MethodGet, urlPath, nil, nil, nil, &res)
		require.NoError(a.t, err)
		actualContent := string(res)
		return actualContent == expectedContent
	}, 30*time.Second, 5*time.Second)
}

func (a *syncTest) remoteNotExist(ctx context.Context, relativePath string) {
	_, err := a.f.Stat(ctx, relativePath)
	require.ErrorIs(a.t, err, fs.ErrNotExist)
}

func (a *syncTest) remoteExists(ctx context.Context, relativePath string) {
	_, err := a.f.Stat(ctx, relativePath)
	require.NoError(a.t, err)
}

func (a *syncTest) touchFile(ctx context.Context, path string) {
	err := a.f.Write(ctx, path, strings.NewReader("contents"), filer.CreateParentDirectories)
	require.NoError(a.t, err)
}

func (a *syncTest) objectType(ctx context.Context, relativePath, expected string) {
	path := path.Join(a.remoteRoot, relativePath)

	a.c.Eventually(func() bool {
		metadata, err := a.w.Workspace.GetStatusByPath(ctx, path)
		if err != nil {
			return false
		}
		return metadata.ObjectType.String() == expected
	}, 30*time.Second, 5*time.Second)
}

func (a *syncTest) language(ctx context.Context, relativePath, expected string) {
	path := path.Join(a.remoteRoot, relativePath)

	a.c.Eventually(func() bool {
		metadata, err := a.w.Workspace.GetStatusByPath(ctx, path)
		if err != nil {
			return false
		}
		return metadata.Language.String() == expected
	}, 30*time.Second, 5*time.Second)
}

func (a *syncTest) snapshotContains(files []string) {
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
		_, ok := s.LastModifiedTimes[filePath]
		assert.True(a.t, ok, "%s not in snapshot file: %v", filePath, s.LastModifiedTimes)
	}
	assert.Equal(a.t, len(files), len(s.LastModifiedTimes), "files=%s s.LastModifiedTimes=%s", files, s.LastModifiedTimes)
}

func TestSyncFullFileSync(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--full", "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.txt"))
	assertSync.remoteFileContent(ctx, "foo.txt", "")

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)
}

func TestSyncIncrementalFileSync(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.txt"))
	assertSync.remoteFileContent(ctx, "foo.txt", "")
	assertSync.snapshotContains(append(repoFiles, "foo.txt"))

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)
	assertSync.snapshotContains(repoFiles)
}

func TestSyncNestedFolderSync(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "dir1/dir2/dir3/foo.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "dir1"))
	assertSync.remoteDirContent(ctx, "dir1", []string{"dir2"})
	assertSync.remoteDirContent(ctx, "dir1/dir2", []string{"dir3"})
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{"foo.txt"})
	assertSync.snapshotContains(append(repoFiles, "dir1/dir2/dir3/foo.txt"))

	// delete
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteNotExist(ctx, "dir1")
	assertSync.snapshotContains(repoFiles)
}

func TestSyncNestedFolderDoesntFailOnNonEmptyDirectory(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "dir1/dir2/dir3/foo.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "dir1/dir2/dir3", []string{"foo.txt"})

	// Add file to dir1 to simulate a user writing to the workspace directly.
	assertSync.touchFile(ctx, "dir1/foo.txt")

	// Remove original file.
	f.Remove(t)
	assertSync.waitForCompletionMarker()

	// Sync should have removed these directories.
	assertSync.remoteNotExist(ctx, "dir1/dir2/dir3")
	assertSync.remoteNotExist(ctx, "dir1/dir2")

	// Sync should have ignored not being able to delete dir1.
	assertSync.remoteExists(ctx, "dir1/foo.txt")
	assertSync.remoteExists(ctx, "dir1")
}

func TestSyncNestedSpacePlusAndHashAreEscapedSync(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "dir1/a b+c/c+d e/e+f g#i.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "dir1"))
	assertSync.remoteDirContent(ctx, "dir1", []string{"a b+c"})
	assertSync.remoteDirContent(ctx, "dir1/a b+c", []string{"c+d e"})
	assertSync.remoteDirContent(ctx, "dir1/a b+c/c+d e", []string{"e+f g#i.txt"})
	assertSync.snapshotContains(append(repoFiles, "dir1/a b+c/c+d e/e+f g#i.txt"))

	// delete
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteNotExist(ctx, "dir1/a b+c/c+d e")
	assertSync.snapshotContains(repoFiles)
}

// This is a check for the edge case when a user does the following:
//
// 1. Add file foo/bar.txt
// 2. Delete foo/bar.txt (including the directory)
// 3. Add file foo
//
// In the above scenario sync should delete the empty folder and add foo to the remote
// file system
func TestSyncIncrementalFileOverwritesFolder(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	// create foo/bar.txt
	localFilePath := filepath.Join(assertSync.localRoot, "foo/bar.txt")
	err := os.MkdirAll(filepath.Dir(localFilePath), 0o755)
	assert.NoError(t, err)
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo"))
	assertSync.remoteDirContent(ctx, "foo", []string{"bar.txt"})
	assertSync.snapshotContains(append(repoFiles, "foo/bar.txt"))

	// delete foo/bar.txt
	f.Remove(t)
	os.Remove(filepath.Join(assertSync.localRoot, "foo"))
	assertSync.waitForCompletionMarker()
	assertSync.remoteNotExist(ctx, "foo")
	assertSync.snapshotContains(repoFiles)

	f2 := testfile.CreateFile(t, filepath.Join(assertSync.localRoot, "foo"))
	defer f2.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo"))
	assertSync.objectType(ctx, "foo", "FILE")
	assertSync.snapshotContains(append(repoFiles, "foo"))
}

func TestSyncIncrementalSyncPythonNotebookToFile(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	// create python notebook
	localFilePath := filepath.Join(assertSync.localRoot, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	f.Overwrite(t, "# Databricks notebook source")

	// notebook was uploaded properly
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo"))
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")
	assertSync.snapshotContains(append(repoFiles, "foo.py"))

	// convert to vanilla python file
	f.Overwrite(t, "# No longer a python notebook")
	assertSync.waitForCompletionMarker()
	assertSync.objectType(ctx, "foo.py", "FILE")
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.py"))
	assertSync.snapshotContains(append(repoFiles, "foo.py"))

	// delete the vanilla python file
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)
	assertSync.snapshotContains(repoFiles)
}

func TestSyncIncrementalSyncFileToPythonNotebook(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	// create vanilla python file
	localFilePath := filepath.Join(assertSync.localRoot, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()

	// assert file upload
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.py"))
	assertSync.objectType(ctx, "foo.py", "FILE")
	assertSync.snapshotContains(append(repoFiles, "foo.py"))

	// convert to notebook
	f.Overwrite(t, "# Databricks notebook source")
	assertSync.waitForCompletionMarker()
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo"))
	assertSync.snapshotContains(append(repoFiles, "foo.py"))
}

func TestSyncIncrementalSyncPythonNotebookDelete(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	// create python notebook
	localFilePath := filepath.Join(assertSync.localRoot, "foo.py")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	f.Overwrite(t, "# Databricks notebook source")
	assertSync.waitForCompletionMarker()

	// notebook was uploaded properly
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo"))
	assertSync.objectType(ctx, "foo", "NOTEBOOK")
	assertSync.language(ctx, "foo", "PYTHON")

	// Delete notebook
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)
}

func TestSyncEnsureRemotePathIsUsableIfRepoDoesntExist(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	wsc := wt.W

	me, err := wsc.CurrentUser.Me(ctx)
	require.NoError(t, err)

	// Hypothetical repo path doesn't exist.
	nonExistingRepoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, testutil.RandomName("doesnt-exist-"))
	err = sync.EnsureRemotePathIsUsable(ctx, wsc, nonExistingRepoPath, nil)
	assert.ErrorContains(t, err, " does not exist; please create it first")

	// Paths nested under a hypothetical repo path should yield the same error.
	nestedPath := path.Join(nonExistingRepoPath, "nested/directory")
	err = sync.EnsureRemotePathIsUsable(ctx, wsc, nestedPath, nil)
	assert.ErrorContains(t, err, " does not exist; please create it first")
}

func TestSyncEnsureRemotePathIsUsableIfRepoExists(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	wsc := wt.W

	_, remoteRepoPath := setupRepo(t, wsc, ctx)

	// Repo itself is usable.
	err := sync.EnsureRemotePathIsUsable(ctx, wsc, remoteRepoPath, nil)
	assert.NoError(t, err)

	// Path nested under repo path is usable.
	nestedPath := path.Join(remoteRepoPath, "nested/directory")
	err = sync.EnsureRemotePathIsUsable(ctx, wsc, nestedPath, nil)
	assert.NoError(t, err)

	// Verify that the directory has been created.
	info, err := wsc.Workspace.GetStatusByPath(ctx, nestedPath)
	require.NoError(t, err)
	require.Equal(t, workspace.ObjectTypeDirectory, info.ObjectType)
}

func TestSyncEnsureRemotePathIsUsableInWorkspace(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	wsc := wt.W

	me, err := wsc.CurrentUser.Me(ctx)
	require.NoError(t, err)

	remotePath := fmt.Sprintf("/Users/%s/%s", me.UserName, testutil.RandomName("ensure-path-exists-test-"))
	err = sync.EnsureRemotePathIsUsable(ctx, wsc, remotePath, me)
	assert.NoError(t, err)

	// Clean up directory after test.
	defer func() {
		err := wsc.Workspace.Delete(ctx, workspace.Delete{
			Path: remotePath,
		})
		assert.NoError(t, err)
	}()

	// Verify that the directory has been created.
	info, err := wsc.Workspace.GetStatusByPath(ctx, remotePath)
	require.NoError(t, err)
	require.Equal(t, workspace.ObjectTypeDirectory, info.ObjectType)
}
