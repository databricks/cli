package bundle

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccFilesAreSyncedCorrectlyWhenNoSnapshot(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	nodeTypeId := internal.GetNodeTypeId(env)
	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})
	require.NoError(t, err)

	t.Setenv("BUNDLE_ROOT", bundleRoot)

	// Add some test file to the bundle
	err = os.WriteFile(filepath.Join(bundleRoot, "test.py"), []byte("print('Hello, World!')"), 0644)
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	remoteRoot := getBundleRemoteRootPath(w, t, uniqueId)

	// Check that test file is in workspace
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "test.py"))
	require.NoError(t, err)

	// Check that deployment.json is synced correctly
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "state", deploy.DeploymentStateFileName))
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment like in CI/CD environment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	// Remove the file from the bundle and deploy again
	err = os.Remove(filepath.Join(bundleRoot, "test.py"))
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// Check that removed file is not in workspace anymore
	_, err = w.Workspace.GetStatusByPath(ctx, path.Join(remoteRoot, "files", "test.py"))
	require.ErrorContains(t, err, "files/test.py")
	require.ErrorContains(t, err, "doesn't exist")
}
