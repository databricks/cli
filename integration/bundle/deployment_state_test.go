package bundle_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFilesAreSyncedCorrectlyWhenNoSnapshot(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)

	// Add some test file to the bundle
	err := os.WriteFile(filepath.Join(bundleRoot, "test.py"), []byte("print('Hello, World!')"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(bundleRoot, "test_to_modify.py"), []byte("print('Hello, World!')"), 0o644)
	require.NoError(t, err)

	// Add notebook to the bundle
	err = os.WriteFile(filepath.Join(bundleRoot, "notebook.py"), []byte("# Databricks notebook source\nHello, World!"), 0o644)
	require.NoError(t, err)

	deployBundle(t, ctx, bundleRoot)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	remoteRoot := getBundleRemoteRootPath(w, t, uniqueId)

	// Check that test file is in workspace
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "test.py"))
	require.NoError(t, err)

	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "test_to_modify.py"))
	require.NoError(t, err)

	// Check that notebook is in workspace
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "notebook"))
	require.NoError(t, err)

	// Check that deployment.json is synced correctly
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "state", deploy.DeploymentStateFileName))
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment like in CI/CD environment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	// Remove the file from the bundle
	err = os.Remove(filepath.Join(bundleRoot, "test.py"))
	require.NoError(t, err)

	// Remove the notebook from the bundle and deploy again
	err = os.Remove(filepath.Join(bundleRoot, "notebook.py"))
	require.NoError(t, err)

	// Modify the content of another file
	err = os.WriteFile(filepath.Join(bundleRoot, "test_to_modify.py"), []byte("print('Modified!')"), 0o644)
	require.NoError(t, err)

	deployBundle(t, ctx, bundleRoot)

	// Check that removed file is not in workspace anymore
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "test.py"))
	require.ErrorContains(t, err, "files/test.py")
	require.ErrorContains(t, err, "doesn't exist")

	// Check that removed notebook is not in workspace anymore
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "notebook"))
	require.ErrorContains(t, err, "files/notebook")
	require.ErrorContains(t, err, "doesn't exist")

	// Check the content of modified file
	content, err := w.Workspace.ReadFile(ctx, path.Join(remoteRoot, "files", "test_to_modify.py"))
	require.NoError(t, err)
	require.Equal(t, "print('Modified!')", string(content))
}
