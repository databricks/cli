package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccEmptyBundleDeploy(t *testing.T) {
	ctx, w := acc.WorkspaceTest(t)

	uniqueId := uuid.New().String()
	me, err := w.W.CurrentUser.Me(ctx)
	require.NoError(t, err)
	remoteRoot := fmt.Sprintf("/Workspace/Users/%s/.bundle/%s", me.UserName, uniqueId)

	// create empty bundle
	tmpDir := t.TempDir()
	f, err := os.Create(filepath.Join(tmpDir, "databricks.yml"))
	require.NoError(t, err)

	bundleRoot := fmt.Sprintf(`bundle:
  name: %s`, uniqueId)
	_, err = f.WriteString(bundleRoot)
	require.NoError(t, err)
	f.Close()

	_, err = w.W.Workspace.GetStatusByPath(ctx, remoteRoot)
	assert.ErrorContains(t, err, "doesn't exist")

	mustValidateBundle(t, ctx, tmpDir)

	// regression: "bundle validate" must not create a directory
	_, err = w.W.Workspace.GetStatusByPath(ctx, remoteRoot)
	require.ErrorContains(t, err, "doesn't exist")

	// deploy empty bundle
	err = deployBundle(t, ctx, tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, tmpDir)
		require.NoError(t, err)
	})

	// verify that remoteRoot was actually relevant location to test
	_, err = w.W.Workspace.GetStatusByPath(ctx, remoteRoot)
	assert.NoError(t, err)

}
