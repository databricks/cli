package internal

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccWorkspaceList(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	stdout, stderr := RequireSuccessfulRun(t, "workspace", "list", "/")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Type")
	assert.Contains(t, outStr, "Language")
	assert.Contains(t, outStr, "Path")
	assert.Equal(t, "", stderr.String())
}

func TestWorkpaceListErrorWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "workspace", "list")
	assert.Equal(t, "accepts 1 arg(s), received 0", err.Error())
}

func TestWorkpaceGetStatusErrorWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "workspace", "get-status")
	assert.Equal(t, "accepts 1 arg(s), received 0", err.Error())
}

func setupWorkspaceImportExportTest(t *testing.T) (context.Context, filer.Filer, string) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryWorkspaceDir(t, w)
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	// Check if we can use this API here, skip test if we cannot.
	_, err = f.Read(ctx, "we_use_this_call_to_test_if_this_api_is_enabled")
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusBadRequest {
		t.Skip(aerr.Message)
	}

	return ctx, f, tmpdir
}

// TODO: add tests for the progress event output logs: https://github.com/databricks/cli/issues/447
func assertFileContents(t *testing.T, path string, content string) {
	require.FileExists(t, path)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), content)
}

func TestAccExportDir(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write test data to the workspace
	err = f.Write(ctx, "file-a", strings.NewReader("abc"))
	require.NoError(t, err)
	err = f.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "sqlNotebook.sql", strings.NewReader("-- Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "scalaNotebook.scala", strings.NewReader("// Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "rNotebook.r", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "a/b/c/file-b", strings.NewReader("def"), filer.CreateParentDirectories)
	require.NoError(t, err)

	// Run Export
	RequireSuccessfulRun(t, "workspace", "export-dir", sourceDir, targetDir)

	// Assert files were exported
	assertFileContents(t, filepath.Join(targetDir, "file-a"), "abc")
	assertFileContents(t, filepath.Join(targetDir, "pyNotebook.py"), "# Databricks notebook source")
	assertFileContents(t, filepath.Join(targetDir, "sqlNotebook.sql"), "-- Databricks notebook source")
	assertFileContents(t, filepath.Join(targetDir, "rNotebook.r"), "# Databricks notebook source")
	assertFileContents(t, filepath.Join(targetDir, "scalaNotebook.scala"), "// Databricks notebook source")
	assertFileContents(t, filepath.Join(targetDir, "a/b/c/file-b"), "def")
}

func TestAccExportDirDoesNotOverwrite(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write remote file
	err = f.Write(ctx, "file-a", strings.NewReader("content from workspace"))
	require.NoError(t, err)

	// Write local file
	err = os.WriteFile(filepath.Join(targetDir, "file-a"), []byte("local content"), os.ModePerm)
	require.NoError(t, err)

	// Run Export
	RequireSuccessfulRun(t, "workspace", "export-dir", sourceDir, targetDir)

	// Assert file is not overwritten
	assertFileContents(t, filepath.Join(targetDir, "file-a"), "local content")
}

func TestAccExportDirWithOverwriteFlag(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write remote file
	err = f.Write(ctx, "file-a", strings.NewReader("content from workspace"))
	require.NoError(t, err)

	// Write local file
	err = os.WriteFile(filepath.Join(targetDir, "file-a"), []byte("local content"), os.ModePerm)
	require.NoError(t, err)

	// Run Export
	RequireSuccessfulRun(t, "workspace", "export-dir", sourceDir, targetDir, "--overwrite")

	// Assert file has been overwritten
	assertFileContents(t, filepath.Join(targetDir, "file-a"), "content from workspace")
}
