package snapshot_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/snapshot"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeBundleWithFiles(t *testing.T, files map[string]string) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	root := vfs.MustNew(dir)
	return &bundle.Bundle{
		BundleRootPath: dir,
		SyncRoot:       root,
		// WorktreeRoot = SyncRoot is the fallback used by LoadGitDetails when
		// there is no git repository.
		WorktreeRoot: root,
		Config:       config.Root{},
	}
}

func TestBundleZipIsDeterministic(t *testing.T) {
	b := makeBundleWithFiles(t, map[string]string{
		"main.py":     "print('hello')",
		"src/task.py": "def run(): pass",
	})

	zip1, err := snapshot.BundleZip(t.Context(), b)
	require.NoError(t, err)
	zip2, err := snapshot.BundleZip(t.Context(), b)
	require.NoError(t, err)

	assert.Equal(t, zip1, zip2, "BundleZip must produce identical bytes for identical content")
}

func TestBundleZipChangesWithContent(t *testing.T) {
	b1 := makeBundleWithFiles(t, map[string]string{"main.py": "v1"})
	b2 := makeBundleWithFiles(t, map[string]string{"main.py": "v2"})

	zip1, err := snapshot.BundleZip(t.Context(), b1)
	require.NoError(t, err)
	zip2, err := snapshot.BundleZip(t.Context(), b2)
	require.NoError(t, err)

	assert.NotEqual(t, zip1, zip2, "different file content must produce different zips")
}

func TestBundleZipRespectsExcludes(t *testing.T) {
	b := makeBundleWithFiles(t, map[string]string{
		"main.py":   "print('hello')",
		"skip.json": `{"id": "runtime-generated"}`,
	})
	bExclude := makeBundleWithFiles(t, map[string]string{
		"main.py":   "print('hello')",
		"skip.json": `{"id": "runtime-generated"}`,
	})
	bExclude.Config.Sync.Exclude = []string{"*.json"}

	zipAll, err := snapshot.BundleZip(t.Context(), b)
	require.NoError(t, err)
	zipExcl, err := snapshot.BundleZip(t.Context(), bExclude)
	require.NoError(t, err)

	// The zip without the excluded file should be smaller and different.
	assert.NotEqual(t, zipAll, zipExcl)
	assert.Less(t, len(zipExcl), len(zipAll))
}

func TestIDFromContent(t *testing.T) {
	id := snapshot.IDFromContent([]byte("hello"))
	// SHA-256 of "hello"
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", id)
	assert.Len(t, id, 64, "SHA-256 hex must be 64 characters")
}

func TestSnapshotPath(t *testing.T) {
	p := snapshot.SnapshotPath("my-bundle", "abc123")
	assert.Equal(t, "/Workspace/Shared/.snapshots/my-bundle/abc123", p)
}

func TestSnapshotIDMatchesBundleZipHash(t *testing.T) {
	b := makeBundleWithFiles(t, map[string]string{"task.py": "x = 1"})

	zipContent, err := snapshot.BundleZip(t.Context(), b)
	require.NoError(t, err)
	expectedID := snapshot.IDFromContent(zipContent)

	id, err := snapshot.SnapshotID(t.Context(), b)
	require.NoError(t, err)

	assert.Equal(t, expectedID, id)
}

func zipEntryNames(t *testing.T, zipContent []byte) []string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	require.NoError(t, err)
	names := make([]string, len(r.File))
	for i, f := range r.File {
		names[i] = f.Name
	}
	return names
}

func TestBundleZipStripsNotebookExtensions(t *testing.T) {
	// Minimal valid Jupyter notebook content.
	ipynb := `{"nbformat": 4, "nbformat_minor": 5, "cells": [], "metadata": {}}`
	b := makeBundleWithFiles(t, map[string]string{
		"src/my_notebook.ipynb": ipynb,
		"src/script.py":         "print('hello')",
	})

	zipContent, err := snapshot.BundleZip(t.Context(), b)
	require.NoError(t, err)

	names := zipEntryNames(t, zipContent)
	assert.True(t, slices.Contains(names, "files/src/my_notebook"), "notebook should have extension stripped")
	assert.False(t, slices.Contains(names, "files/src/my_notebook.ipynb"), "notebook should not appear with .ipynb extension")
	assert.True(t, slices.Contains(names, "files/src/script.py"), "regular Python file should keep its extension")
}
