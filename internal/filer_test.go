package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filerTest struct {
	*testing.T
	filer.Filer
}

func (f filerTest) assertContents(ctx context.Context, name string, contents string) {
	reader, err := f.Read(ctx, name)
	if !assert.NoError(f, err) {
		return
	}

	defer reader.Close()

	var body bytes.Buffer
	_, err = io.Copy(&body, reader)
	if !assert.NoError(f, err) {
		return
	}

	assert.Equal(f, contents, body.String())
}

// Common tests we run across all filers to ensure consistent behavior.
func commonFilerReadWriteTests(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	// Read should fail because the root path doesn't yet exist.
	_, err = f.Read(ctx, "/foo/bar")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Read should fail because the path points to a directory
	err = f.Mkdir(ctx, "/dir")
	require.NoError(t, err)
	_, err = f.Read(ctx, "/dir")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Write with CreateParentDirectories flag should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`), filer.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello world`)

	// Write should fail because there is an existing file at the specified path.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`))
	assert.True(t, errors.As(err, &filer.FileAlreadyExistsError{}))
	assert.True(t, errors.Is(err, fs.ErrExist))

	// Write with OverwriteIfExists should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`), filer.OverwriteIfExists)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello universe`)

	// Write should succeed if there is no existing file at the specified path.
	err = f.Write(ctx, "/foo/qux", strings.NewReader(`hello universe`))
	assert.NoError(t, err)

	// Stat on a directory should succeed.
	// Note: size and modification time behave differently between backends.
	info, err := f.Stat(ctx, "/foo")
	require.NoError(t, err)
	assert.Equal(t, "foo", info.Name())
	assert.True(t, info.Mode().IsDir())
	assert.Equal(t, true, info.IsDir())

	// Stat on a file should succeed.
	// Note: size and modification time behave differently between backends.
	info, err = f.Stat(ctx, "/foo/bar")
	require.NoError(t, err)
	assert.Equal(t, "bar", info.Name())
	assert.True(t, info.Mode().IsRegular())
	assert.Equal(t, false, info.IsDir())

	// Delete should fail if the file doesn't exist.
	err = f.Delete(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Stat should fail if the file doesn't exist.
	_, err = f.Stat(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Delete should succeed for file that does exist.
	err = f.Delete(ctx, "/foo/bar")
	assert.NoError(t, err)

	// Delete should fail for a non-empty directory.
	err = f.Delete(ctx, "/foo")
	assert.True(t, errors.As(err, &filer.DirectoryNotEmptyError{}))
	assert.True(t, errors.Is(err, fs.ErrInvalid))

	// Delete of the filer root should ALWAYS fail, otherwise subsequent writes would fail.
	// It is not in the filer's purview to delete its root directory.
	err = f.Delete(ctx, "/")
	assert.True(t, errors.As(err, &filer.CannotDeleteRootError{}))
	assert.True(t, errors.Is(err, fs.ErrInvalid))
}

// Write should fail because the intermediate directory doesn't exist.
func writeDoesNotCreateIntermediateDirs(t *testing.T, ctx context.Context, f filer.Filer) {
	err := f.Write(ctx, "/this-does-not-exist/bar", strings.NewReader(`hello world`))
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

// Write should succeed even if the intermediate directory doesn't exist.
func writeCreatesIntermediateDirs(t *testing.T, ctx context.Context, f filer.Filer) {
	err := f.Write(ctx, "/this-does-not-exist/bar", strings.NewReader(`hello world`), filer.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/this-does-not-exist/bar", `hello world`)
}

// Delete should succeed for a directory with files in it if the DeleteRecursively flag is set.
func recursiveDeleteSupported(t *testing.T, ctx context.Context, f filer.Filer) {
	err := f.Mkdir(ctx, "/dir-to-delete")
	require.NoError(t, err)

	err = f.Write(ctx, "/dir-to-delete/file1", strings.NewReader(`hello world`))
	require.NoError(t, err)

	err = f.Delete(ctx, "/dir-to-delete", filer.DeleteRecursively)
	assert.NoError(t, err)
}

// Delete should fail if the DeleteRecursively flag is set.
func recursiveDeleteNotSupported(t *testing.T, ctx context.Context, f filer.Filer) {
	err := f.Mkdir(ctx, "/dir-to-delete")
	require.NoError(t, err)

	err = f.Write(ctx, "/dir-to-delete/file1", strings.NewReader(`hello world`))
	require.NoError(t, err)

	err = f.Delete(ctx, "/dir-to-delete", filer.DeleteRecursively)
	assert.ErrorContains(t, err, "files API does not support recursive delete")
}

func TestAccFilerReadWrite(t *testing.T) {
	t.Parallel()

	t.Run("local", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		f, _ := setupLocalFiler(t)

		// Tests specific to the Local API.
		writeDoesNotCreateIntermediateDirs(t, ctx, f)
		recursiveDeleteSupported(t, ctx, f)

		// Common tests we run across all filers to ensure consistent behavior.
		commonFilerReadWriteTests(t, ctx, f)
	})

	t.Run("workspace files", func(t *testing.T) {
		t.Parallel()
		ctx, f := setupWsfsFiler(t)

		// Tests specific to the Workspace Files API.
		writeDoesNotCreateIntermediateDirs(t, ctx, f)
		recursiveDeleteSupported(t, ctx, f)

		// Common tests we run across all filers to ensure consistent behavior.
		commonFilerReadWriteTests(t, ctx, f)
	})

	t.Run("dbfs", func(t *testing.T) {
		t.Parallel()
		f, _ := setupDbfsFiler(t)
		ctx := context.Background()

		// Tests specific to the DBFS API.
		writeDoesNotCreateIntermediateDirs(t, ctx, f)
		recursiveDeleteSupported(t, ctx, f)

		// Common tests we run across all filers to ensure consistent behavior.
		commonFilerReadWriteTests(t, ctx, f)
	})

	t.Run("files", func(t *testing.T) {
		t.Parallel()
		f, _ := setupUcVolumesFiler(t)
		ctx := context.Background()

		// Tests specific to the Files API.
		writeCreatesIntermediateDirs(t, ctx, f)
		recursiveDeleteNotSupported(t, ctx, f)

		// Common tests we run across all filers to ensure consistent behavior.
		commonFilerReadWriteTests(t, ctx, f)
	})
}

// Common tests we run across all filers to ensure consistent behavior.
func commonFilerReadDirTest(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error
	var info fs.FileInfo

	// We start with an empty directory.
	entries, err := f.ReadDir(ctx, ".")
	require.NoError(t, err)
	assert.Len(t, entries, 0)

	// Write a file.
	err = f.Write(ctx, "/hello.txt", strings.NewReader(`hello world`))
	require.NoError(t, err)

	// Create a directory.
	err = f.Mkdir(ctx, "/dir")
	require.NoError(t, err)

	// Write a file.
	err = f.Write(ctx, "/dir/world.txt", strings.NewReader(`hello world`))
	require.NoError(t, err)

	// Create a nested directory (check that it creates intermediate directories).
	err = f.Mkdir(ctx, "/dir/a/b/c")
	require.NoError(t, err)

	// Expect an error if the path doesn't exist.
	_, err = f.ReadDir(ctx, "/dir/a/b/c/d/e")
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}), err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Expect two entries in the root.
	entries, err = f.ReadDir(ctx, ".")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "dir", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "hello.txt", entries[1].Name())
	assert.False(t, entries[1].IsDir())
	info, err = entries[1].Info()
	require.NoError(t, err)
	assert.Greater(t, info.ModTime().Unix(), int64(0))

	// Expect two entries in the directory.
	entries, err = f.ReadDir(ctx, "/dir")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "a", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "world.txt", entries[1].Name())
	assert.False(t, entries[1].IsDir())
	info, err = entries[1].Info()
	require.NoError(t, err)
	assert.Greater(t, info.ModTime().Unix(), int64(0))

	// Expect a single entry in the nested path.
	entries, err = f.ReadDir(ctx, "/dir/a/b")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "c", entries[0].Name())
	assert.True(t, entries[0].IsDir())

	// Expect an error trying to call ReadDir on a file
	_, err = f.ReadDir(ctx, "/hello.txt")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Expect 0 entries for an empty directory
	err = f.Mkdir(ctx, "empty-dir")
	require.NoError(t, err)
	entries, err = f.ReadDir(ctx, "empty-dir")
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Expect one entry for a directory with a file in it
	err = f.Write(ctx, "dir-with-one-file/my-file.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	entries, err = f.ReadDir(ctx, "dir-with-one-file")
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entries[0].Name(), "my-file.txt")
	assert.False(t, entries[0].IsDir())
}

func TestAccFilerReadDir(t *testing.T) {
	t.Parallel()

	t.Run("local", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		f, _ := setupLocalFiler(t)

		commonFilerReadDirTest(t, ctx, f)
	})

	t.Run("workspace files", func(t *testing.T) {
		t.Parallel()
		ctx, f := setupWsfsFiler(t)

		commonFilerReadDirTest(t, ctx, f)
	})

	t.Run("dbfs", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		f, _ := setupDbfsFiler(t)

		commonFilerReadDirTest(t, ctx, f)
	})

	t.Run("files", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		f, tmpdir := setupUcVolumesFiler(t)

		fmt.Println(tmpdir)

		commonFilerReadDirTest(t, ctx, f)
	})
}

var jupyterNotebookContent1 = `
{
	"cells": [
	 {
	  "cell_type": "code",
	  "execution_count": null,
	  "metadata": {},
	  "outputs": [],
	  "source": [
	   "print(\"Jupyter Notebook Version 1\")"
	  ]
	 }
	],
	"metadata": {
	 "language_info": {
	  "name": "python"
	 },
	 "orig_nbformat": 4
	},
	"nbformat": 4,
	"nbformat_minor": 2
   }
`

var jupyterNotebookContent2 = `
{
	"cells": [
	 {
	  "cell_type": "code",
	  "execution_count": null,
	  "metadata": {},
	  "outputs": [],
	  "source": [
	   "print(\"Jupyter Notebook Version 2\")"
	  ]
	 }
	],
	"metadata": {
	 "language_info": {
	  "name": "python"
	 },
	 "orig_nbformat": 4
	},
	"nbformat": 4,
	"nbformat_minor": 2
   }
`

func TestAccFilerWorkspaceNotebookConflict(t *testing.T) {
	ctx, f := setupWsfsFiler(t)
	var err error

	// Upload the notebooks
	err = f.Write(ctx, "pyNb.py", strings.NewReader("# Databricks notebook source\nprint('first upload'))"))
	require.NoError(t, err)
	err = f.Write(ctx, "rNb.r", strings.NewReader("# Databricks notebook source\nprint('first upload'))"))
	require.NoError(t, err)
	err = f.Write(ctx, "sqlNb.sql", strings.NewReader("-- Databricks notebook source\n SELECT \"first upload\""))
	require.NoError(t, err)
	err = f.Write(ctx, "scalaNb.scala", strings.NewReader("// Databricks notebook source\n println(\"first upload\"))"))
	require.NoError(t, err)
	err = f.Write(ctx, "jupyterNb.ipynb", strings.NewReader(jupyterNotebookContent1))
	require.NoError(t, err)

	// Assert contents after initial upload
	filerTest{t, f}.assertContents(ctx, "pyNb", "# Databricks notebook source\nprint('first upload'))")
	filerTest{t, f}.assertContents(ctx, "rNb", "# Databricks notebook source\nprint('first upload'))")
	filerTest{t, f}.assertContents(ctx, "sqlNb", "-- Databricks notebook source\n SELECT \"first upload\"")
	filerTest{t, f}.assertContents(ctx, "scalaNb", "// Databricks notebook source\n println(\"first upload\"))")
	filerTest{t, f}.assertContents(ctx, "jupyterNb", "# Databricks notebook source\nprint(\"Jupyter Notebook Version 1\")")

	// Assert uploading a second time fails due to overwrite mode missing
	err = f.Write(ctx, "pyNb.py", strings.NewReader("# Databricks notebook source\nprint('second upload'))"))
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.Regexp(t, regexp.MustCompile(`file already exists: .*/pyNb$`), err.Error())

	err = f.Write(ctx, "rNb.r", strings.NewReader("# Databricks notebook source\nprint('second upload'))"))
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.Regexp(t, regexp.MustCompile(`file already exists: .*/rNb$`), err.Error())

	err = f.Write(ctx, "sqlNb.sql", strings.NewReader("# Databricks notebook source\n SELECT \"second upload\")"))
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.Regexp(t, regexp.MustCompile(`file already exists: .*/sqlNb$`), err.Error())

	err = f.Write(ctx, "scalaNb.scala", strings.NewReader("# Databricks notebook source\n println(\"second upload\"))"))
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.Regexp(t, regexp.MustCompile(`file already exists: .*/scalaNb$`), err.Error())

	err = f.Write(ctx, "jupyterNb.ipynb", strings.NewReader(jupyterNotebookContent2))
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.Regexp(t, regexp.MustCompile(`file already exists: .*/jupyterNb$`), err.Error())
}

func TestAccFilerWorkspaceNotebookWithOverwriteFlag(t *testing.T) {
	ctx, f := setupWsfsFiler(t)
	var err error

	// Upload notebooks
	err = f.Write(ctx, "pyNb.py", strings.NewReader("# Databricks notebook source\nprint('first upload'))"))
	require.NoError(t, err)
	err = f.Write(ctx, "rNb.r", strings.NewReader("# Databricks notebook source\nprint('first upload'))"))
	require.NoError(t, err)
	err = f.Write(ctx, "sqlNb.sql", strings.NewReader("-- Databricks notebook source\n SELECT \"first upload\""))
	require.NoError(t, err)
	err = f.Write(ctx, "scalaNb.scala", strings.NewReader("// Databricks notebook source\n println(\"first upload\"))"))
	require.NoError(t, err)
	err = f.Write(ctx, "jupyterNb.ipynb", strings.NewReader(jupyterNotebookContent1))
	require.NoError(t, err)

	// Assert contents after initial upload
	filerTest{t, f}.assertContents(ctx, "pyNb", "# Databricks notebook source\nprint('first upload'))")
	filerTest{t, f}.assertContents(ctx, "rNb", "# Databricks notebook source\nprint('first upload'))")
	filerTest{t, f}.assertContents(ctx, "sqlNb", "-- Databricks notebook source\n SELECT \"first upload\"")
	filerTest{t, f}.assertContents(ctx, "scalaNb", "// Databricks notebook source\n println(\"first upload\"))")
	filerTest{t, f}.assertContents(ctx, "jupyterNb", "# Databricks notebook source\nprint(\"Jupyter Notebook Version 1\")")

	// Upload notebooks a second time, overwriting the initial uplaods
	err = f.Write(ctx, "pyNb.py", strings.NewReader("# Databricks notebook source\nprint('second upload'))"), filer.OverwriteIfExists)
	require.NoError(t, err)
	err = f.Write(ctx, "rNb.r", strings.NewReader("# Databricks notebook source\nprint('second upload'))"), filer.OverwriteIfExists)
	require.NoError(t, err)
	err = f.Write(ctx, "sqlNb.sql", strings.NewReader("-- Databricks notebook source\n SELECT \"second upload\""), filer.OverwriteIfExists)
	require.NoError(t, err)
	err = f.Write(ctx, "scalaNb.scala", strings.NewReader("// Databricks notebook source\n println(\"second upload\"))"), filer.OverwriteIfExists)
	require.NoError(t, err)
	err = f.Write(ctx, "jupyterNb.ipynb", strings.NewReader(jupyterNotebookContent2), filer.OverwriteIfExists)
	require.NoError(t, err)

	// Assert contents have been overwritten
	filerTest{t, f}.assertContents(ctx, "pyNb", "# Databricks notebook source\nprint('second upload'))")
	filerTest{t, f}.assertContents(ctx, "rNb", "# Databricks notebook source\nprint('second upload'))")
	filerTest{t, f}.assertContents(ctx, "sqlNb", "-- Databricks notebook source\n SELECT \"second upload\"")
	filerTest{t, f}.assertContents(ctx, "scalaNb", "// Databricks notebook source\n println(\"second upload\"))")
	filerTest{t, f}.assertContents(ctx, "jupyterNb", "# Databricks notebook source\nprint(\"Jupyter Notebook Version 2\")")
}
