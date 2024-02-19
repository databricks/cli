package internal

import (
	"bytes"
	"context"
	"errors"
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

func commonFilerRecursiveDeleteTest(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	err = f.Write(ctx, "dir/file1", strings.NewReader("content1"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/file1", `content1`)

	err = f.Write(ctx, "dir/file2", strings.NewReader("content2"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/file2", `content2`)

	err = f.Write(ctx, "dir/subdir1/file3", strings.NewReader("content3"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/subdir1/file3", `content3`)

	err = f.Write(ctx, "dir/subdir1/file4", strings.NewReader("content4"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/subdir1/file4", `content4`)

	err = f.Write(ctx, "dir/subdir2/file5", strings.NewReader("content5"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/subdir2/file5", `content5`)

	err = f.Write(ctx, "dir/subdir2/file6", strings.NewReader("content6"), filer.CreateParentDirectories)
	require.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "dir/subdir2/file6", `content6`)

	entriesBeforeDelete, err := f.ReadDir(ctx, "dir")
	require.NoError(t, err)
	assert.Len(t, entriesBeforeDelete, 4)

	names := []string{}
	for _, e := range entriesBeforeDelete {
		names = append(names, e.Name())
	}
	assert.Equal(t, names, []string{"file1", "file2", "subdir1", "subdir2"})

	err = f.Delete(ctx, "dir")
	assert.ErrorAs(t, err, &filer.DirectoryNotEmptyError{})

	err = f.Delete(ctx, "dir", filer.DeleteRecursively)
	assert.NoError(t, err)
	_, err = f.ReadDir(ctx, "dir")
	assert.ErrorAs(t, err, &filer.NoSuchDirectoryError{})
}

func TestAccFilerRecursiveDelete(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		f    func(t *testing.T) (filer.Filer, string)
	}{
		{"local", setupLocalFiler},
		{"workspace files", setupWsfsFiler},
		{"dbfs", setupDbfsFiler},
		{"files", setupUcVolumesFiler},
	} {
		tc := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f, _ := tc.f(t)
			ctx := context.Background()

			// Common tests we run across all filers to ensure consistent behavior.
			commonFilerRecursiveDeleteTest(t, ctx, f)
		})
	}
}

// Common tests we run across all filers to ensure consistent behavior.
func commonFilerReadWriteTests(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	// Write should fail because the intermediate directory doesn't exist.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`))
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Read should fail because the intermediate directory doesn't yet exist.
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

	// Delete should succeed for a non-empty directory if the DeleteRecursively flag is set.
	err = f.Delete(ctx, "/foo", filer.DeleteRecursively)
	assert.NoError(t, err)

	// Delete of the filer root should ALWAYS fail, otherwise subsequent writes would fail.
	// It is not in the filer's purview to delete its root directory.
	err = f.Delete(ctx, "/")
	assert.True(t, errors.As(err, &filer.CannotDeleteRootError{}))
	assert.True(t, errors.Is(err, fs.ErrInvalid))
}

func TestAccFilerReadWrite(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		f    func(t *testing.T) (filer.Filer, string)
	}{
		{"local", setupLocalFiler},
		{"workspace files", setupWsfsFiler},
		{"dbfs", setupDbfsFiler},
		{"files", setupUcVolumesFiler},
	} {
		tc := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f, _ := tc.f(t)
			ctx := context.Background()

			// Common tests we run across all filers to ensure consistent behavior.
			commonFilerReadWriteTests(t, ctx, f)
		})
	}
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

	for _, testCase := range []struct {
		name string
		f    func(t *testing.T) (filer.Filer, string)
	}{
		{"local", setupLocalFiler},
		{"workspace files", setupWsfsFiler},
		{"dbfs", setupDbfsFiler},
		{"files", setupUcVolumesFiler},
	} {
		tc := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f, _ := tc.f(t)
			ctx := context.Background()

			commonFilerReadDirTest(t, ctx, f)
		})
	}
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
	f, _ := setupWsfsFiler(t)
	ctx := context.Background()
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
	f, _ := setupWsfsFiler(t)
	ctx := context.Background()
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
