package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSourceDir(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	err = f.Write(ctx, "pyNb.py", strings.NewReader("# Databricks notebook source\nprint(123)"))
	require.NoError(t, err)

	err = f.Write(ctx, "query.sql", strings.NewReader("SELECT 1"))
	require.NoError(t, err)

	err = f.Write(ctx, "a/b/c/hello.txt", strings.NewReader("hello, world\n"), filer.CreateParentDirectories)
	require.NoError(t, err)
}

func setupSourceFile(t *testing.T, ctx context.Context, f filer.Filer) {
	err := f.Write(ctx, "foo.txt", strings.NewReader("abc"))
	require.NoError(t, err)
}

func assertTargetFile(t *testing.T, ctx context.Context, f filer.Filer, relPath string) {
	var err error

	r, err := f.Read(ctx, relPath)
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(b))
}

func assertFileContent(t *testing.T, ctx context.Context, f filer.Filer, path, expectedContent string) {
	r, err := f.Read(ctx, path)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(b))
}

func assertTargetDir(t *testing.T, ctx context.Context, f filer.Filer) {
	assertFileContent(t, ctx, f, "pyNb.py", "# Databricks notebook source\nprint(123)")
	assertFileContent(t, ctx, f, "query.sql", "SELECT 1")
	assertFileContent(t, ctx, f, "a/b/c/hello.txt", "hello, world\n")
}

func setupLocalFiler(t *testing.T) (filer.Filer, string) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmp := t.TempDir()
	f, err := filer.NewLocalClient(tmp)
	require.NoError(t, err)
	return f, "file:" + tmp
}

func setupDbfsFiler(t *testing.T) (filer.Filer, string) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)
	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)
	return f, "dbfs:" + tmpDir
}

type cpTest struct {
	setupSource func(*testing.T) (filer.Filer, string)
	setupTarget func(*testing.T) (filer.Filer, string)
}

func setupTable() []cpTest {
	return []cpTest{
		{setupSource: setupLocalFiler, setupTarget: setupLocalFiler},
		{setupSource: setupLocalFiler, setupTarget: setupDbfsFiler},
		{setupSource: setupDbfsFiler, setupTarget: setupLocalFiler},
		{setupSource: setupDbfsFiler, setupTarget: setupDbfsFiler},
	}
}

func TestFsCpDir(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		RequireSuccessfulRun(t, "fs", "cp", "-r", sourceDir, targetDir)

		assertTargetDir(t, ctx, targetFiler)
	}
}

func TestFsCpFileToFile(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceFile(t, ctx, sourceFiler)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "foo.txt"), path.Join(targetDir, "bar.txt"))

		assertTargetFile(t, ctx, targetFiler, "bar.txt")
	}
}

func TestFsCpFileToDir(t *testing.T) {
	ctx := context.Background()
	table := setupTable()
	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceFile(t, ctx, sourceFiler)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "foo.txt"), targetDir)

		assertTargetFile(t, ctx, targetFiler, "foo.txt")
	}
}

func TestFsCpDirToDirFileNotOverwritten(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", sourceDir, targetDir, "--recursive")
		assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
		assertFileContent(t, ctx, targetFiler, "query.sql", "SELECT 1")
		assertFileContent(t, ctx, targetFiler, "pyNb.py", "# Databricks notebook source\nprint(123)")
	}
}

func TestFsCpFileToDirFileNotOverwritten(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"))
		assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
	}
}

func TestFsCpFileToFileFileNotOverwritten(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hola.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/hola.txt"), "--recursive")
		assertFileContent(t, ctx, targetFiler, "a/b/c/hola.txt", "this should not be overwritten")
	}
}

func TestFsCpDirToDirWithOverwriteFlag(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this will be overwritten"), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", sourceDir, targetDir, "--recursive", "--overwrite")
		assertTargetDir(t, ctx, targetFiler)
	}
}

func TestFsCpFileToFileWithOverwriteFlag(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hola.txt", strings.NewReader("this will be overwritten. Such is life."), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/hola.txt"), "--overwrite")
		assertFileContent(t, ctx, targetFiler, "a/b/c/hola.txt", "hello, world\n")
	}
}

func TestFsCpFileToDirWithOverwriteFlag(t *testing.T) {
	ctx := context.Background()
	table := setupTable()

	for _, row := range table {
		sourceFiler, sourceDir := row.setupSource(t)
		targetFiler, targetDir := row.setupTarget(t)
		setupSourceDir(t, ctx, sourceFiler)

		// Write a conflicting file to target
		err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this will be overwritten :') "), filer.CreateParentDirectories)
		require.NoError(t, err)

		RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"), "--recursive", "--overwrite")
		assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "hello, world\n")
	}
}

func TestFsCpErrorsWhenSourceIsDirWithoutRecursiveFlag(t *testing.T) {
	type test struct {
		path   string
		scheme string
	}

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	table := []test{
		{path: temporaryDbfsDir(t, w), scheme: "dbfs"},
		{path: t.TempDir(), scheme: "file"},
	}

	for _, row := range table {
		_, _, err := RequireErrorRun(t, "fs", "cp", row.scheme+":"+row.path, "file:/a")
		assert.Equal(t, fmt.Sprintf("source path %s is a directory. Please specify the --recursive flag", row.path), err.Error())
	}
}

// TODO: test out all the error cases
// TODO: Test cp works for relative local paths

func TestFsCpErrorsOnNoScheme(t *testing.T) {
	_, _, err := RequireErrorRun(t, "fs", "cp", "/a", "/b")
	assert.Equal(t, "no scheme specified for path /a. Please specify scheme \"dbfs\" or \"file\". Example: file:/foo/bar", err.Error())
}

func TestFsCpErrorsOnInvalidScheme(t *testing.T) {
	_, _, err := RequireErrorRun(t, "fs", "cp", "file:/a", "https:/b")
	assert.Equal(t, "unsupported scheme https specified for path https:/b. Please specify scheme \"dbfs\" or \"file\". Example: file:/foo/bar", err.Error())
}

func TestFsCpSourceIsDirectoryButTargetIsFile(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupDbfsFiler(t)
	targetFiler, targetDir := setupDbfsFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	// Write a conflicting file to target
	err := targetFiler.Write(ctx, "my_target", strings.NewReader("I'll block any attempts to recursively copy"), filer.CreateParentDirectories)
	require.NoError(t, err)

	_, _, err = RequireErrorRun(t, "fs", "cp", sourceDir, path.Join(targetDir, "my_target"), "--recursive", "--overwrite")
	assert.Regexp(t, regexp.MustCompile(`Cannot create directory .* because .* is an existing file.`), err.Error())
}

func TestFsCpSourceIsDirectoryButTargetIsLocalFile(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupDbfsFiler(t)
	targetFiler, targetDir := setupLocalFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	// Write a conflicting file to target
	err := targetFiler.Write(ctx, "my_target", strings.NewReader("I'll block any attempts to recursively copy"), filer.CreateParentDirectories)
	require.NoError(t, err)

	_, _, err = RequireErrorRun(t, "fs", "cp", sourceDir, path.Join(targetDir, "my_target"), "--recursive", "--overwrite")
	assert.Regexp(t, regexp.MustCompile(`mkdir .*: not a directory`), err.Error())
}
