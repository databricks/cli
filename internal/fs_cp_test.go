package internal

import (
	"context"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

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
	var err error

	r, err := f.Read(ctx, relPath)
	assert.NoError(t, err)
	defer r.Close()
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(b))
}

func assertFileContent(t *testing.T, ctx context.Context, f filer.Filer, path, expectedContent string) {
	r, err := f.Read(ctx, path)
	require.NoError(t, err)
	defer r.Close()
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(b))
}

func assertTargetDir(t *testing.T, ctx context.Context, f filer.Filer) {
	assertFileContent(t, ctx, f, "pyNb.py", "# Databricks notebook source\nprint(123)")
	assertFileContent(t, ctx, f, "query.sql", "SELECT 1")
	assertFileContent(t, ctx, f, "a/b/c/hello.txt", "hello, world\n")
}

type cpTest struct {
	name        string
	setupSource func(*testing.T) (filer.Filer, string)
	setupTarget func(*testing.T) (filer.Filer, string)
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

func TestAccFsCpDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			RequireSuccessfulRun(t, "fs", "cp", sourceDir, targetDir, "--recursive")

			assertTargetDir(t, context.Background(), targetFiler)
		})
	}
}

func TestAccFsCpFileToFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceFile(t, context.Background(), sourceFiler)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "foo.txt"), path.Join(targetDir, "bar.txt"))

			assertTargetFile(t, context.Background(), targetFiler, "bar.txt")
		})
	}
}

func TestAccFsCpFileToDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceFile(t, context.Background(), sourceFiler)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "foo.txt"), targetDir)

			assertTargetFile(t, context.Background(), targetFiler, "foo.txt")
		})
	}
}

func TestAccFsCpFileToDirForWindowsPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping test on non-windows OS")
	}

	ctx := context.Background()
	sourceFiler, sourceDir := setupLocalFiler(t)
	targetFiler, targetDir := setupDbfsFiler(t)
	setupSourceFile(t, ctx, sourceFiler)

	windowsPath := filepath.Join(filepath.FromSlash(sourceDir), "foo.txt")

	RequireSuccessfulRun(t, "fs", "cp", windowsPath, targetDir)
	assertTargetFile(t, ctx, targetFiler, "foo.txt")
}

func TestAccFsCpDirToDirFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", sourceDir, targetDir, "--recursive")
			assertFileContent(t, context.Background(), targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
			assertFileContent(t, context.Background(), targetFiler, "query.sql", "SELECT 1")
			assertFileContent(t, context.Background(), targetFiler, "pyNb.py", "# Databricks notebook source\nprint(123)")
		})
	}
}

func TestAccFsCpFileToDirFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/hello.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"))
			assertFileContent(t, context.Background(), targetFiler, "a/b/c/hello.txt", "this should not be overwritten")
		})
	}
}

func TestAccFsCpFileToFileFileNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/dontoverwrite.txt", strings.NewReader("this should not be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/dontoverwrite.txt"))
			assertFileContent(t, context.Background(), targetFiler, "a/b/c/dontoverwrite.txt", "this should not be overwritten")
		})
	}
}

func TestAccFsCpDirToDirWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/hello.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", sourceDir, targetDir, "--recursive", "--overwrite")
			assertTargetDir(t, context.Background(), targetFiler)
		})
	}
}

func TestAccFsCpFileToFileWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/overwritten.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c/overwritten.txt"), "--overwrite")
			assertFileContent(t, context.Background(), targetFiler, "a/b/c/overwritten.txt", "hello, world\n")
		})
	}
}

func TestAccFsCpFileToDirWithOverwriteFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "a/b/c/hello.txt", strings.NewReader("this should be overwritten"), filer.CreateParentDirectories)
			require.NoError(t, err)

			RequireSuccessfulRun(t, "fs", "cp", path.Join(sourceDir, "a/b/c/hello.txt"), path.Join(targetDir, "a/b/c"), "--overwrite")
			assertFileContent(t, context.Background(), targetFiler, "a/b/c/hello.txt", "hello, world\n")
		})
	}
}

func TestAccFsCpErrorsWhenSourceIsDirWithoutRecursiveFlag(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tmpDir := tc.setupFiler(t)

			_, _, err := RequireErrorRun(t, "fs", "cp", path.Join(tmpDir), path.Join(tmpDir, "foobar"))
			r := regexp.MustCompile("source path .* is a directory. Please specify the --recursive flag")
			assert.Regexp(t, r, err.Error())
		})
	}
}

func TestAccFsCpErrorsOnInvalidScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cp", "dbfs:/a", "https:/b")
	assert.Equal(t, "invalid scheme: https", err.Error())
}

func TestAccFsCpSourceIsDirectoryButTargetIsFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range copyTests() {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceFiler, sourceDir := tc.setupSource(t)
			targetFiler, targetDir := tc.setupTarget(t)
			setupSourceDir(t, context.Background(), sourceFiler)

			// Write a conflicting file to target
			err := targetFiler.Write(context.Background(), "my_target", strings.NewReader("I'll block any attempts to recursively copy"), filer.CreateParentDirectories)
			require.NoError(t, err)

			_, _, err = RequireErrorRun(t, "fs", "cp", sourceDir, path.Join(targetDir, "my_target"), "--recursive")
			assert.Error(t, err)
		})
	}
}
