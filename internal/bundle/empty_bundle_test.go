package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccEmptyBundleDeploy(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	// create empty bundle
	tmpDir := t.TempDir()
	f, err := os.Create(filepath.Join(tmpDir, "databricks.yml"))
	require.NoError(t, err)

	bundleRoot := fmt.Sprintf(`bundle:
  name: %s`, uuid.New().String())
	_, err = f.WriteString(bundleRoot)
	require.NoError(t, err)
	f.Close()

	// deploy empty bundle
	err = deployBundle(t, ctx, tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, tmpDir)
		require.NoError(t, err)
	})
}
