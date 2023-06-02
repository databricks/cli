package internal

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/sync/repofiles"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type repofilesTestHelper struct {
	w   *databricks.WorkspaceClient
	f   filer.Filer
	ctx context.Context
	t   *testing.T

	localRoot  string
	remoteRoot string
}

func setupRepofilesTestHelper(t *testing.T, ctx context.Context) *repofilesTestHelper {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// initialize client
	wsfsTmpDir := temporaryWorkspaceDir(t, w)
	localTmpDir := t.TempDir()

	require.NoError(t, err)
	f, err := filer.NewWorkspaceFilesClient(w, wsfsTmpDir)
	require.NoError(t, err)

	return &repofilesTestHelper{
		w:   w,
		f:   f,
		ctx: ctx,
		t:   t,

		localRoot:  localTmpDir,
		remoteRoot: wsfsTmpDir,
	}
}

func (h *repofilesTestHelper) createLocalFile(name string, content string) {
	absPath := filepath.Join(h.localRoot, name)
	err := os.MkdirAll(filepath.Dir(absPath), os.ModePerm)
	require.NoError(h.t, err)
	err = os.WriteFile(absPath, []byte(content), os.ModePerm)
	require.NoError(h.t, err)
}

func (h *repofilesTestHelper) createRemoteFile(name string, content string) {
	h.f.Write(h.ctx, name, strings.NewReader(content), filer.CreateParentDirectories)
}

func (h *repofilesTestHelper) createRemoteDirectory(name string) {
	h.f.Mkdir(h.ctx, name)
}

func (h *repofilesTestHelper) assertRemoteFileContent(name string, content string) {
	assertFileContains(h.t, h.ctx, h.f, name, content)
}

func (h *repofilesTestHelper) assertRemoteFileType(name string, fileType workspace.ObjectType) {
	info, err := h.f.Stat(h.ctx, name)
	require.NoError(h.t, err)

	objectInfo := info.Sys().(workspace.ObjectInfo)
	assert.Equal(h.t, fileType, objectInfo.ObjectType)
}

func TestRepoFilesPutFile(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)

	// create local file
	helper.createLocalFile("foo.txt", "hello, world")
	err = r.PutFile(ctx, "foo.txt")
	require.NoError(t, err)

	// Expect PUT to succeed
	helper.assertRemoteFileContent("foo.txt", "hello, world")
}

func TestRepoFilesPutFileOverwritesNotebook(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)

	// Create notebook in workspace
	helper.createRemoteFile("foo.py", "#Databricks notebook source\nprint(1)")
	helper.assertRemoteFileType("foo", workspace.ObjectTypeNotebook)

	// Put file and assert file PUT succeeded
	helper.createLocalFile("foo", "this file will overwrite the notebook")
	err = r.PutFile(ctx, "foo")
	assert.NoError(t, err)
	helper.assertRemoteFileContent("foo", "this file will overwrite the notebook")
	helper.assertRemoteFileType("foo", workspace.ObjectTypeFile)
}

func TestRepoFilesPutFileOverwritesEmptyDirectoryTree(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)

	// create empty remote directory tree
	helper.createRemoteDirectory("foo/a/b/c")
	helper.createRemoteDirectory("foo/a/b/d/e")
	helper.createRemoteDirectory("foo/f/g/i")

	// assert directory tree is created
	helper.assertRemoteFileType("foo", workspace.ObjectTypeDirectory)
	helper.assertRemoteFileType("foo/a/b/c", workspace.ObjectTypeDirectory)
	helper.assertRemoteFileType("foo/f/g/i", workspace.ObjectTypeDirectory)
	helper.assertRemoteFileType("foo/a/b/d/e", workspace.ObjectTypeDirectory)

	// Create local file and PUT it into the workspace
	helper.createLocalFile("foo", "hello, world")
	err = r.PutFile(ctx, "foo")
	require.NoError(t, err)
	helper.assertRemoteFileContent("foo", "hello, world")
	helper.assertRemoteFileType("foo", workspace.ObjectTypeFile)
}

func TestRepoFilesPutFileInDirOverwritesExistingNotebook(t *testing.T) {
	// TODO: Skipping this test for now since the workspace-files import API has a
	// bug and does not return the error message we need
	t.SkipNow()

	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)

	// create remote notebook
	helper.createRemoteFile("foo.py", "#Databricks notebook source\nprint(1)")
	helper.assertRemoteFileType("foo", workspace.ObjectTypeNotebook)

	// create local file and PUT it in the workspace
	helper.createLocalFile("foo/hello.txt", "just a file")
	err = r.PutFile(ctx, "foo/hello.txt")
	require.NoError(t, err)

	// Assert PUT succeeeded
	helper.assertRemoteFileType("foo", workspace.ObjectTypeDirectory)
	helper.assertRemoteFileContent("foo/bar.txt", "just a file")
}

func TestRepoFilesPutFileWithoutOverwrite(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: false,
	})
	require.NoError(t, err)

	// create local file
	helper.createLocalFile("foo.txt", "hello, world")
	err = r.PutFile(ctx, "foo.txt")
	require.NoError(t, err)

	// Expect PUT to succeed
	helper.assertRemoteFileContent("foo.txt", "hello, world")
}

func TestRepoFilesPutFileWithoutOverwriteFails(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: false,
	})
	require.NoError(t, err)

	// create remote file
	helper.createRemoteFile("foo.txt", "this file already exists in the workspace")

	// create local file
	helper.createLocalFile("foo.txt", "this file will attempt to overwrite the workspace file and fail")

	// assert overwrite fails
	err = r.PutFile(ctx, "foo.txt")
	assert.ErrorIs(t, err, fs.ErrExist)
}

func TestRepoFilesPutFileWithoutOverwriteFailsIfDirectoryExists(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: false,
	})
	require.NoError(t, err)

	helper.createRemoteDirectory("foo")

	// create local file
	helper.createLocalFile("foo", "hello, world")
	err = r.PutFile(ctx, "foo")

	// Assert PUT failed because file already exists
	assert.ErrorIs(t, err, fs.ErrExist)
}

func TestRepoFilesPutFileWithoutOverwriteFailsIfNotebookExists(t *testing.T) {
	ctx := context.Background()
	helper := setupRepofilesTestHelper(t, ctx)

	r, err := repofiles.Create(helper.remoteRoot, helper.localRoot, helper.w, &repofiles.RepoFileOptions{
		OverwriteIfExists: false,
	})
	require.NoError(t, err)

	// create remote notebook
	helper.createRemoteFile("foo.py", "#Databricks notebook source\nprint(1)")

	// create local file
	helper.createLocalFile("foo", "hello, world")
	err = r.PutFile(ctx, "foo")

	// Assert PUT failed because file already exists
	assert.ErrorIs(t, err, fs.ErrExist)
}
