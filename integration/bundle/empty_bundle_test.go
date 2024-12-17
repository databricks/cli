package bundle_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmptyBundleDeploy(t *testing.T) {
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
	deployBundle(t, ctx, tmpDir)

	t.Cleanup(func() {
		destroyBundle(t, ctx, tmpDir)
	})
}
