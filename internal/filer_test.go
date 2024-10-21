package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path"
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

func (f filerTest) assertContentsJupyter(ctx context.Context, name string, language string) {
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

	var actual map[string]any
	err = json.Unmarshal(body.Bytes(), &actual)
	if !assert.NoError(f, err) {
		return
	}

	// Since a roundtrip to the workspace changes a Jupyter notebook's payload,
	// the best we can do is assert that the nbformat is correct.
	assert.EqualValues(f, 4, actual["nbformat"])
	assert.Equal(f, language, actual["metadata"].(map[string]any)["language_info"].(map[string]any)["name"])
}

func (f filerTest) assertNotExists(ctx context.Context, name string) {
	_, err := f.Stat(ctx, name)
	assert.ErrorIs(f, err, fs.ErrNotExist)
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
		{"workspace files extensions", setupWsfsExtensionsFiler},
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
	assert.ErrorAs(t, err, &filer.FileDoesNotExistError{})
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Stat should fail if the file doesn't exist.
	_, err = f.Stat(ctx, "/doesnt_exist")
	assert.ErrorAs(t, err, &filer.FileDoesNotExistError{})
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Delete should succeed for file that does exist.
	err = f.Delete(ctx, "/foo/bar")
	assert.NoError(t, err)

	// Delete should fail for a non-empty directory.
	err = f.Delete(ctx, "/foo")
	assert.ErrorAs(t, err, &filer.DirectoryNotEmptyError{})
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
		{"workspace files extensions", setupWsfsExtensionsFiler},
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
		{"workspace files extensions", setupWsfsExtensionsFiler},
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

// TODO: Continue adding tests for other types of notebooks than python here.
// Exporting from a workspace makes this work easier.

func TestAccFilerWorkspaceNotebookConflict(t *testing.T) {
	t.Parallel()

	f, _ := setupWsfsFiler(t)
	ctx := context.Background()
	var err error

	for _, tcases := range []struct {
		name           string
		nameWithoutExt string
		content1       string
		expected1      string
		content2       string
	}{
		{
			name:           "pyNb.py",
			nameWithoutExt: "pyNb",
			content1:       "# Databricks notebook source\nprint('first upload'))",
			expected1:      "# Databricks notebook source\nprint('first upload'))",
			content2:       "# Databricks notebook source\nprint('second upload'))",
		},
		{
			name:           "rNb.r",
			nameWithoutExt: "rNb",
			content1:       "# Databricks notebook source\nprint('first upload'))",
			expected1:      "# Databricks notebook source\nprint('first upload'))",
			content2:       "# Databricks notebook source\nprint('second upload'))",
		},
		{
			name:           "sqlNb.sql",
			nameWithoutExt: "sqlNb",
			content1:       "-- Databricks notebook source\n SELECT \"first upload\"",
			expected1:      "-- Databricks notebook source\n SELECT \"first upload\"",
			content2:       "-- Databricks notebook source\n SELECT \"second upload\"",
		},
		{
			name:           "scalaNb.scala",
			nameWithoutExt: "scalaNb",
			content1:       "// Databricks notebook source\n println(\"first upload\"))",
			expected1:      "// Databricks notebook source\n println(\"first upload\"))",
			content2:       "// Databricks notebook source\n println(\"second upload\"))",
		},
		{
			name:           "pythonJupyterNb.ipynb",
			nameWithoutExt: "pythonJupyterNb",
			content1:       readFile(t, "testdata/notebooks/py1.ipynb"),
			expected1:      "# Databricks notebook source\nprint(1)",
			content2:       readFile(t, "testdata/notebooks/py2.ipynb"),
		},
		{
			name:           "rJupyterNb.ipynb",
			nameWithoutExt: "rJupyterNb",
			content1:       readFile(t, "testdata/notebooks/r1.ipynb"),
			expected1:      "# Databricks notebook source\nprint(1)",
			content2:       readFile(t, "testdata/notebooks/r2.ipynb"),
		},
	} {
		t.Run(tcases.name, func(t *testing.T) {
			// Upload the notebook
			err = f.Write(ctx, tcases.name, strings.NewReader(tcases.content1))
			require.NoError(t, err)

			// Assert contents after initial upload
			filerTest{t, f}.assertContents(ctx, tcases.nameWithoutExt, tcases.expected1)

			// Assert uploading a second time fails due to overwrite mode missing
			err = f.Write(ctx, tcases.name, strings.NewReader(tcases.content2))
			assert.ErrorIs(t, err, fs.ErrExist)
			assert.Regexp(t, regexp.MustCompile(`file already exists: .*/`+tcases.nameWithoutExt+`$`), err.Error())

		})
	}
}

func TestAccFilerWorkspaceNotebookWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	f, _ := setupWsfsFiler(t)
	ctx := context.Background()
	var err error

	for _, tcases := range []struct {
		name           string
		nameWithoutExt string
		content1       string
		expected1      string
		content2       string
		expected2      string
	}{
		{
			name:           "pyNb.py",
			nameWithoutExt: "pyNb",
			content1:       "# Databricks notebook source\nprint('first upload'))",
			expected1:      "# Databricks notebook source\nprint('first upload'))",
			content2:       "# Databricks notebook source\nprint('second upload'))",
			expected2:      "# Databricks notebook source\nprint('second upload'))",
		},
		{
			name:           "rNb.r",
			nameWithoutExt: "rNb",
			content1:       "# Databricks notebook source\nprint('first upload'))",
			expected1:      "# Databricks notebook source\nprint('first upload'))",
			content2:       "# Databricks notebook source\nprint('second upload'))",
			expected2:      "# Databricks notebook source\nprint('second upload'))",
		},
		{
			name:           "sqlNb.sql",
			nameWithoutExt: "sqlNb",
			content1:       "-- Databricks notebook source\n SELECT \"first upload\"",
			expected1:      "-- Databricks notebook source\n SELECT \"first upload\"",
			content2:       "-- Databricks notebook source\n SELECT \"second upload\"",
			expected2:      "-- Databricks notebook source\n SELECT \"second upload\"",
		},
		{
			name:           "scalaNb.scala",
			nameWithoutExt: "scalaNb",
			content1:       "// Databricks notebook source\n println(\"first upload\"))",
			expected1:      "// Databricks notebook source\n println(\"first upload\"))",
			content2:       "// Databricks notebook source\n println(\"second upload\"))",
			expected2:      "// Databricks notebook source\n println(\"second upload\"))",
		},
		{
			name:           "pythonJupyterNb.ipynb",
			nameWithoutExt: "pythonJupyterNb",
			content1:       readFile(t, "testdata/notebooks/py1.ipynb"),
			expected1:      "# Databricks notebook source\nprint(1))",
			content2:       readFile(t, "testdata/notebooks/py2.ipynb"),
			expected2:      "# Databricks notebook source\nprint(2))",
		},
		{
			name:           "rJupyterNb.ipynb",
			nameWithoutExt: "rJupyterNb",
			content1:       readFile(t, "testdata/notebooks/r1.ipynb"),
			expected1:      "# Databricks notebook source\nprint(1))",
			content2:       readFile(t, "testdata/notebooks/r2.ipynb"),
			expected2:      "# Databricks notebook source\nprint(2))",
		},
	} {
		t.Run(tcases.name, func(t *testing.T) {
			// Upload the notebook
			err = f.Write(ctx, tcases.name, strings.NewReader(tcases.content1))
			require.NoError(t, err)

			// Assert contents after initial upload
			filerTest{t, f}.assertContents(ctx, tcases.nameWithoutExt, tcases.expected1)

			// Upload the notebook a second time with overwrite flag
			err = f.Write(ctx, tcases.name, strings.NewReader(tcases.content2), filer.OverwriteIfExists)
			require.NoError(t, err)

			// Assert contents after second upload
			filerTest{t, f}.assertContents(ctx, tcases.nameWithoutExt, tcases.expected2)
		})
	}
}

// TODO: Add a test that the exported file has the right extension / language_info set?
// Required for DABs in the workspace.
func TestAccFilerWorkspaceFilesExtensionsReadDir(t *testing.T) {
	t.Parallel()

	files := []struct {
		name    string
		content string
	}{
		{"dir1/dir2/dir3/file.txt", "file content"},
		{"dir1/notebook.py", "# Databricks notebook source\nprint('first upload'))"},
		{"foo.py", "print('foo')"},
		{"foo.r", "print('foo')"},
		{"foo.scala", "println('foo')"},
		{"foo.sql", "SELECT 'foo'"},
		{"py1.ipynb", readFile(t, "testdata/notebooks/py1.ipynb")},
		{"py2.ipynb", readFile(t, "testdata/notebooks/py2.ipynb")},
		{"r1.ipynb", readFile(t, "testdata/notebooks/r1.ipynb")},
		{"pyNb.py", "# Databricks notebook source\nprint('first upload'))"},
		{"rNb.r", "# Databricks notebook source\nprint('first upload'))"},
		{"scalaNb.scala", "// Databricks notebook source\n println(\"first upload\"))"},
		{"sqlNb.sql", "-- Databricks notebook source\n SELECT \"first upload\""},
	}

	// Assert that every file has a unique basename
	basenames := map[string]struct{}{}
	for _, f := range files {
		basename := path.Base(f.name)
		if _, ok := basenames[basename]; ok {
			t.Fatalf("basename %s is not unique", basename)
		}
		basenames[basename] = struct{}{}
	}

	ctx := context.Background()
	wf, _ := setupWsfsExtensionsFiler(t)

	for _, f := range files {
		err := wf.Write(ctx, f.name, strings.NewReader(f.content), filer.CreateParentDirectories)
		require.NoError(t, err)
	}

	// Read entries
	entries, err := wf.ReadDir(ctx, ".")
	require.NoError(t, err)
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{
		"dir1",
		"foo.py",
		"foo.r",
		"foo.scala",
		"foo.sql",
		"py1.ipynb",
		"py2.ipynb",
		"pyNb.py",
		"r1.ipynb",
		"rNb.r",
		"scalaNb.scala",
		"sqlNb.sql",
	}, names)

	// Read entries in subdirectory
	entries, err = wf.ReadDir(ctx, "dir1")
	require.NoError(t, err)
	names = []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{
		"dir2",
		"notebook.py",
	}, names)
}

func setupFilerWithExtensionsTest(t *testing.T) filer.Filer {
	files := []struct {
		name    string
		content string
	}{
		{"foo.py", "# Databricks notebook source\nprint('first upload'))"},
		{"bar.py", "print('foo')"},
		{"p1.ipynb", readFile(t, "testdata/notebooks/py1.ipynb")},
		{"r1.ipynb", readFile(t, "testdata/notebooks/r1.ipynb")},
		{"pretender", "not a notebook"},
		{"dir/file.txt", "file content"},
		{"scala-notebook.scala", "// Databricks notebook source\nprintln('first upload')"},
	}

	ctx := context.Background()
	wf, _ := setupWsfsExtensionsFiler(t)

	for _, f := range files {
		err := wf.Write(ctx, f.name, strings.NewReader(f.content), filer.CreateParentDirectories)
		require.NoError(t, err)
	}

	return wf
}

func TestAccFilerWorkspaceFilesExtensionsRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wf := setupFilerWithExtensionsTest(t)

	// Read contents of test fixtures as a sanity check.
	filerTest{t, wf}.assertContents(ctx, "foo.py", "# Databricks notebook source\nprint('first upload'))")
	filerTest{t, wf}.assertContents(ctx, "bar.py", "print('foo')")
	filerTest{t, wf}.assertContentsJupyter(ctx, "p1.ipynb", "python")
	filerTest{t, wf}.assertContentsJupyter(ctx, "r1.ipynb", "R")
	filerTest{t, wf}.assertContents(ctx, "dir/file.txt", "file content")
	filerTest{t, wf}.assertContents(ctx, "scala-notebook.scala", "// Databricks notebook source\nprintln('first upload')")
	filerTest{t, wf}.assertContents(ctx, "pretender", "not a notebook")

	// Read non-existent file
	_, err := wf.Read(ctx, "non-existent.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Ensure we do not read a regular file as a notebook
	_, err = wf.Read(ctx, "pretender.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)
	_, err = wf.Read(ctx, "pretender.ipynb")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Read directory
	_, err = wf.Read(ctx, "dir")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Ensure we do not read a Scala notebook as a Python notebook
	_, err = wf.Read(ctx, "scala-notebook.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFilerWorkspaceFilesExtensionsDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wf := setupFilerWithExtensionsTest(t)

	// Delete notebook
	err := wf.Delete(ctx, "foo.py")
	require.NoError(t, err)
	filerTest{t, wf}.assertNotExists(ctx, "foo.py")

	// Delete file
	err = wf.Delete(ctx, "bar.py")
	require.NoError(t, err)
	filerTest{t, wf}.assertNotExists(ctx, "bar.py")

	// Delete python jupyter notebook
	err = wf.Delete(ctx, "p1.ipynb")
	require.NoError(t, err)
	filerTest{t, wf}.assertNotExists(ctx, "p1.ipynb")

	// Delete r jupyter notebook
	err = wf.Delete(ctx, "r1.ipynb")
	require.NoError(t, err)
	filerTest{t, wf}.assertNotExists(ctx, "r1.ipynb")

	// Delete non-existent file
	err = wf.Delete(ctx, "non-existent.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Ensure we do not delete a file as a notebook
	err = wf.Delete(ctx, "pretender.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Ensure we do not delete a Scala notebook as a Python notebook
	_, err = wf.Read(ctx, "scala-notebook.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Delete directory
	err = wf.Delete(ctx, "dir")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Delete directory recursively
	err = wf.Delete(ctx, "dir", filer.DeleteRecursively)
	require.NoError(t, err)
	filerTest{t, wf}.assertNotExists(ctx, "dir")
}

func TestAccFilerWorkspaceFilesExtensionsStat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wf := setupFilerWithExtensionsTest(t)

	// Stat on a notebook
	info, err := wf.Stat(ctx, "foo.py")
	require.NoError(t, err)
	assert.Equal(t, "foo.py", info.Name())
	assert.False(t, info.IsDir())

	// Stat on a file
	info, err = wf.Stat(ctx, "bar.py")
	require.NoError(t, err)
	assert.Equal(t, "bar.py", info.Name())
	assert.False(t, info.IsDir())

	// Stat on a python jupyter notebook
	info, err = wf.Stat(ctx, "p1.ipynb")
	require.NoError(t, err)
	assert.Equal(t, "p1.ipynb", info.Name())
	assert.False(t, info.IsDir())

	// Stat on an R jupyter notebook
	info, err = wf.Stat(ctx, "r1.ipynb")
	require.NoError(t, err)
	assert.Equal(t, "r1.ipynb", info.Name())
	assert.False(t, info.IsDir())

	// Stat on a directory
	info, err = wf.Stat(ctx, "dir")
	require.NoError(t, err)
	assert.Equal(t, "dir", info.Name())
	assert.True(t, info.IsDir())

	// Stat on a non-existent file
	_, err = wf.Stat(ctx, "non-existent.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Ensure we do not stat a file as a notebook
	_, err = wf.Stat(ctx, "pretender.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Ensure we do not stat a Scala notebook as a Python notebook
	_, err = wf.Stat(ctx, "scala-notebook.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	_, err = wf.Stat(ctx, "pretender.ipynb")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccWorkspaceFilesExtensionsDirectoriesAreNotNotebooks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wf, _ := setupWsfsExtensionsFiler(t)

	// Create a directory with an extension
	err := wf.Mkdir(ctx, "foo")
	require.NoError(t, err)

	// Reading foo.py should fail. foo is a directory, not a notebook.
	_, err = wf.Read(ctx, "foo.py")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccWorkspaceFilesExtensions_ExportFormatIsPreserved(t *testing.T) {
	t.Parallel()

	// Case 1: Writing source notebooks.
	for _, tc := range []struct {
		language       string
		sourceName     string
		sourceContent  string
		jupyterName    string
		jupyterContent string
	}{
		{
			language:      "python",
			sourceName:    "foo.py",
			sourceContent: "# Databricks notebook source\nprint('foo')",
			jupyterName:   "foo.ipynb",
		},
		{
			language:      "r",
			sourceName:    "foo.r",
			sourceContent: "# Databricks notebook source\nprint('foo')",
			jupyterName:   "foo.ipynb",
		},
	} {
		t.Run(tc.language, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			wf, _ := setupWsfsExtensionsFiler(t)

			err := wf.Write(ctx, tc.sourceName, strings.NewReader(tc.sourceContent))
			require.NoError(t, err)

			// The source notebook should exist but not the Jupyter notebook
			filerTest{t, wf}.assertContents(ctx, tc.sourceName, tc.sourceContent)
			_, err = wf.Stat(ctx, tc.jupyterName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			_, err = wf.Read(ctx, tc.jupyterName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			err = wf.Delete(ctx, tc.jupyterName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}

	// Case 2: Writing Jupyter notebooks.
	for _, tc := range []struct {
		language       string
		sourceName     string
		jupyterName    string
		jupyterContent string
	}{
		{
			language:       "python",
			sourceName:     "bar.py",
			jupyterName:    "foo.ipynb",
			jupyterContent: readFile(t, "testdata/notebooks/py1.ipynb"),
		},
		{
			language:       "R",
			sourceName:     "bar.r",
			jupyterName:    "foo.ipynb",
			jupyterContent: readFile(t, "testdata/notebooks/r1.ipynb"),
		},
	} {
		t.Run(tc.language, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			wf, _ := setupWsfsExtensionsFiler(t)

			err := wf.Write(ctx, tc.jupyterName, strings.NewReader(tc.jupyterContent))
			require.NoError(t, err)

			// The Jupyter notebook should exist but not the source notebook
			filerTest{t, wf}.assertContentsJupyter(ctx, tc.jupyterName, tc.language)
			_, err = wf.Stat(ctx, tc.sourceName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			_, err = wf.Read(ctx, tc.sourceName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
			err = wf.Delete(ctx, tc.sourceName)
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}
