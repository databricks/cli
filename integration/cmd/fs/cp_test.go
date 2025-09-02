package fs_test

import (
	"context"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
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
	assertFileContent(t, ctx, f, relPath, "abc")
}

func assertFileContent(t *testing.T, ctx context.Context, f filer.Filer, path, expectedContent string) {
	r, err := f.Read(ctx, path)
	if err != nil {
		assert.NoError(t, err)
		return
	}

	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		assert.NoError(t, err)
		return
	}

	assert.Equal(t, expectedContent, string(b))
}

func assertTargetDir(t *testing.T, ctx context.Context, f filer.Filer) {
	assertFileContent(t, ctx, f, "pyNb.py", "# Databricks notebook source\nprint(123)")
	assertFileContent(t, ctx, f, "query.sql", "SELECT 1")
	assertFileContent(t, ctx, f, "a/b/c/hello.txt", "hello, world\n")
}

type cpTest struct {
	name        string
	setupSource func(testutil.TestingT) (filer.Filer, string)
	setupTarget func(testutil.TestingT) (filer.Filer, string)
}

func copyTests() []cpTest {
	return []cpTest{
		// source: local file system
		{
			name:        "local to local",
			setupSource: setupLocalFiler,
			setupTarget: setupLocalFiler,
		},
		{
			name:        "local to dbfs",
			setupSource: setupLocalFiler,
			setupTarget: setupDbfsFiler,
		},
		{
			name:        "local to uc-volumes",
			setupSource: setupLocalFiler,
			setupTarget: setupUcVolumesFiler,
		},

		// source: dbfs
		{
			name:        "dbfs to local",
			setupSource: setupDbfsFiler,
			setupTarget: setupLocalFiler,
		},
		{
			name:        "dbfs to dbfs",
			setupSource: setupDbfsFiler,
			setupTarget: setupDbfsFiler,
		},
		{
			name:        "dbfs to uc-volumes",
			setupSource: setupDbfsFiler,
			setupTarget: setupUcVolumesFiler,
		},

		// source: uc-volumes
		{
			name:        "uc-volumes to local",
			setupSource: setupUcVolumesFiler,
			setupTarget: setupLocalFiler,
		},
		{
			name:        "uc-volumes to dbfs",
			setupSource: setupUcVolumesFiler,
			setupTarget: setupDbfsFiler,
		},
		{
			name:        "uc-volumes to uc-volumes",
			setupSource: setupUcVolumesFiler,
			setupTarget: setupUcVolumesFiler,
		},
	}
}

func TestFsCpDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", sourceDir, targetDir, "--recursive")

			assertTargetDir(t, ctx, targetFiler)
		})
	}
}

func TestFsCpFileToFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceFile(t, ctx, sourceFiler)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "foo.txt"), path.Join(targetDir, "bar.txt"))
			assertTargetFile(t, ctx, targetFiler, "bar.txt")
		})
	}
}

func TestFsCpFileToDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceFile(t, ctx, sourceFiler)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "foo.txt"), targetDir)

			assertTargetFile(t, ctx, targetFiler, "foo.txt")
		})
	}
}

func TestFsCpFileToDirForWindowsPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping test on non-windows OS")
	}

	ctx := context.Background()
	sourceFiler, sourceDir := setupLocalFiler(t)
	targetFiler, targetDir := setupDbfsFiler(t)
	setupSourceFile(t, ctx, sourceFiler)

	windowsPath := filepath.Join(filepath.FromSlash(sourceDir), "foo.txt")

	testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", windowsPath, targetDir)
	assertTargetFile(t, ctx, targetFiler, "foo.txt")
}

func TestFsCpDirToDirFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", sourceDir, targetDir, "--recursive")
			assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
			assertFileContent(t, ctx, targetFiler, "query.sql", "SELECT 1")
			assertFileContent(t, ctx, targetFiler, "pyNb.py", "# Databricks notebook source\nprint(123)")
		})
	}
}

func TestFsCpFileToDirFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"))
			assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
		})
	}
}

func TestFsCpFileToFileFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/dontoverwrite.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/dontoverwrite.txt"))
			assertFileContent(t, ctx, targetFiler, "a/b/c/dontoverwrite.txt", "this should not be overwritten")
		})
	}
}

func TestFsCpDirToDirWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", sourceDir, targetDir, "--recursive", "--overwrite")
			assertTargetDir(t, ctx, targetFiler)
		})
	}
}

func TestFsCpFileToFileWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/overwritten.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/overwritten.txt"), "--overwrite")
			assertFileContent(t, ctx, targetFiler, "a/b/c/overwritten.txt", "hello, world\n")
		})
	}
}

func TestFsCpFileToDirWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "a/b/c/hello.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			testcli.RequireSuccessfulRun(t, ctx, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"), "--overwrite")
			assertFileContent(t, ctx, targetFiler, "a/b/c/hello.txt", "hello, world\n")
		})
	}
}

func TestFsCpErrorsWhenSourceIsDirWithoutRecursiveFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, tmpDir := testCase.setupFiler(t)

			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "cp", path.Join(tmpDir), path.Join(tmpDir, "foobar"))
			r := regexp.MustCompile("source path .* is a directory. Please specify the --recursive flag")
			assert.Regexp(t, r, err.Error())
		})
	}
}

func TestFsCpErrorsOnInvalidScheme(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "cp", "dbfs:/a", "https:/b")
	assert.Equal(t, "invalid scheme: https", err.Error())
}

func TestFsCpSourceIsDirectoryButTargetIsFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sourceFiler, sourceDir := testCase.setupSource(t)
			targetFiler, targetDir := testCase.setupTarget(t)
			setupSourceDir(t, ctx, sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(ctx, "my_target", strings.NewReader("I'll block any attempts to recursively copy"), filer.CreateParentDirectories)
			require.NoError(t, err)

			_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "cp", sourceDir, path.Join(targetDir, "my_target"), "--recursive")
			assert.Error(t, err)
		})
	}
}
