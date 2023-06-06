package internal

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/sync/repofiles"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: skip if not cloud env, these are integration tests
// TODO: split into a separate PR

func TestRepoFilesPutFile(t *testing.T) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	ctx := context.Background()

	// initialize client
	wsfsTmpDir := temporaryWorkspaceDir(t, w)
	localTmpDir := t.TempDir()
	r, err := repofiles.Create(wsfsTmpDir, localTmpDir, w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)
	f, err := filer.NewWorkspaceFilesClient(w, wsfsTmpDir)
	require.NoError(t, err)

	// create local file
	err = os.WriteFile(filepath.Join(localTmpDir, "foo.txt"), []byte(`hello, world`), os.ModePerm)
	require.NoError(t, err)
	err = r.PutFile(ctx, "foo.txt")
	require.NoError(t, err)

	require.NoError(t, f.Mkdir(ctx, "bar"))

	entries, err := f.ReadDir(ctx, "bar")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assertFileContains(t, ctx, f, "foo.txt", "hello, world")
}

func TestRepoFilesFileOverwritesNotebook(t *testing.T) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	ctx := context.Background()

	// initialize client
	wsfsTmpDir := temporaryWorkspaceDir(t, w)
	localTmpDir := t.TempDir()
	r, err := repofiles.Create(wsfsTmpDir, localTmpDir, w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)
	f, err := filer.NewWorkspaceFilesClient(w, wsfsTmpDir)
	require.NoError(t, err)

	// create local notebook
	err = os.WriteFile(filepath.Join(localTmpDir, "foo.py"), []byte("#Databricks notebook source\nprint(1)"), os.ModePerm)
	require.NoError(t, err)

	// upload notebook
	err = r.PutFile(ctx, "foo.py")
	require.NoError(t, err)
	assertNotebookExists(t, ctx, w, path.Join(wsfsTmpDir, "foo"))

	// upload file, and assert that it overwrites the notebook
	err = os.WriteFile(filepath.Join(localTmpDir, "foo"), []byte("I am going to overwrite the notebook"), os.ModePerm)
	require.NoError(t, err)
	err = r.PutFile(ctx, "foo")
	require.NoError(t, err)
	assertFileContains(t, ctx, f, "foo", "I am going to overwrite the notebook")
}

func TestRepoFilesFileOverwritesEmptyDirectoryTree(t *testing.T) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	ctx := context.Background()

	// initialize client
	wsfsTmpDir := temporaryWorkspaceDir(t, w)
	localTmpDir := t.TempDir()
	r, err := repofiles.Create(wsfsTmpDir, localTmpDir, w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)
	f, err := filer.NewWorkspaceFilesClient(w, wsfsTmpDir)
	require.NoError(t, err)

	// create local file
	err = os.WriteFile(filepath.Join(localTmpDir, "foo"), []byte(`hello, world`), os.ModePerm)
	require.NoError(t, err)

	// construct a directory tree without files in the workspace
	err = f.Mkdir(ctx, "foo/a/b/c")
	require.NoError(t, err)
	err = f.Mkdir(ctx, "foo/a/b/d/e")
	require.NoError(t, err)
	err = f.Mkdir(ctx, "foo/f/g/i")
	require.NoError(t, err)

	// assert the directories exist
	entries, err := f.ReadDir(ctx, "foo")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.True(t, entries[0].IsDir())
	assert.True(t, entries[1].IsDir())

	// upload file, and assert that it overwrites the empty directories
	err = r.PutFile(ctx, "foo")
	require.NoError(t, err)
	assertFileContains(t, ctx, f, "foo", "hello, world")

	// assert the directories do not exist anymore
	_, err = f.ReadDir(ctx, "foo")
	assert.ErrorIs(t, err, filer.ErrNotADirectory)
}

func TestRepoFilesFileInDirOverwritesExistingNotebook(t *testing.T) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	ctx := context.Background()

	// initialize client
	wsfsTmpDir := temporaryWorkspaceDir(t, w)
	localTmpDir := t.TempDir()
	r, err := repofiles.Create(wsfsTmpDir, localTmpDir, w, &repofiles.RepoFileOptions{
		OverwriteIfExists: true,
	})
	require.NoError(t, err)
	f, err := filer.NewWorkspaceFilesClient(w, wsfsTmpDir)
	require.NoError(t, err)

	// create local notebook
	err = f.Write(ctx, "foo.py", strings.NewReader("#Databricks notebook source\nprint(1)"))
	require.NoError(t, err)
	assertNotebookExists(t, ctx, w, path.Join(wsfsTmpDir, "foo"))

	// upload file
	err = os.Mkdir(filepath.Join(localTmpDir, "foo"), os.ModePerm)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(localTmpDir, "foo/bar.txt"), []byte("I am going to overwrite the notebook"), os.ModePerm)
	require.NoError(t, err)
	err = r.PutFile(ctx, "foo/bar.txt")
	require.NoError(t, err)
	assertFileContains(t, ctx, f, "foo/bar.txt", "I am going to overwrite the notebook")
}
