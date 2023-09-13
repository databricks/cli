package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/stretchr/testify/require"
)

func TestAccEmptyBundleDeploy(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	tmpDir := t.TempDir()
	f, err := os.Create(filepath.Join(tmpDir, "databricks.yml"))
	require.NoError(t, err)

	_, err = f.WriteString(
		`bundle:
  name: abc`)
	require.NoError(t, err)

	err = deployBundle(t, tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, tmpDir)
		require.NoError(t, err)
	})
}
