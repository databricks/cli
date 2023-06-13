package internal

import (
	"context"
	"io/ioutil"
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

func assertTargetDir(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	r, err := f.Read(ctx, "pyNb.py")
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "# Databricks notebook source\nprint(123)", string(b))

	r, err = f.Read(ctx, "query.sql")
	assert.NoError(t, err)
	b, err = ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1", string(b))

	r, err = f.Read(ctx, "a/b/c/hello.txt")
	assert.NoError(t, err)
	b, err = ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "hello, world\n", string(b))
}

func setupLocalFiler(t *testing.T) (filer.Filer, string) {
	tmp := t.TempDir()
	f, err := filer.NewLocalClient(tmp)
	require.NoError(t, err)
	return f, "file:" + tmp
}

func setupDbfsFiler(t *testing.T) (filer.Filer, string) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)
	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)
	return f, "dbfs:" + tmpDir
}

func TestFsCpDirLocalToDbfs(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupLocalFiler(t)
	targetFiler, targetDir := setupDbfsFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	RequireSuccessfulRun(t, "fs", "cp", "-r", sourceDir, targetDir)

	assertTargetDir(t, ctx, targetFiler)
}

func TestFsCpDirDbfsToDbfs(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupDbfsFiler(t)
	targetFiler, targetDir := setupDbfsFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	RequireSuccessfulRun(t, "fs", "cp", "-r", sourceDir, targetDir)

	assertTargetDir(t, ctx, targetFiler)
}

func TestFsCpDirDbfsToLocal(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupDbfsFiler(t)
	targetFiler, targetDir := setupLocalFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	RequireSuccessfulRun(t, "fs", "cp", "-r", sourceDir, targetDir)

	assertTargetDir(t, ctx, targetFiler)
}

func TestFsCpDirLocalToLocal(t *testing.T) {
	ctx := context.Background()
	sourceFiler, sourceDir := setupLocalFiler(t)
	targetFiler, targetDir := setupLocalFiler(t)
	setupSourceDir(t, ctx, sourceFiler)

	RequireSuccessfulRun(t, "fs", "cp", "-r", sourceDir, targetDir)

	assertTargetDir(t, ctx, targetFiler)
}
