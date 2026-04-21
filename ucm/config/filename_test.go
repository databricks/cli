package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileNames_FindSingle(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte("ucm:\n  name: x\n"), 0o644))

	path, err := config.FileNames.FindInPath(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ucm.yml"), path)
}

func TestFileNames_RejectAmbiguous(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yaml"), []byte(""), 0o644))

	_, err := config.FileNames.FindInPath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple")
}

func TestFileNames_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := config.FileNames.FindInPath(dir)
	require.Error(t, err)
}
