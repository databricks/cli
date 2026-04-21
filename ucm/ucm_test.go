package ucm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FindsUcmYmlUnderPath(t *testing.T) {
	dir := t.TempDir()
	yaml := []byte(`
ucm:
  name: tree-loaded
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), yaml, 0o644))

	u, err := ucm.Load(t.Context(), dir)
	require.NoError(t, err)
	assert.Equal(t, "tree-loaded", u.Config.Ucm.Name)
	assert.Equal(t, filepath.Clean(dir), u.RootPath)
}

func TestLoad_MissingUcmYml(t *testing.T) {
	dir := t.TempDir()
	_, err := ucm.Load(t.Context(), dir)
	require.Error(t, err)
}
