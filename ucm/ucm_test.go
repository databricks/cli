package ucm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
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

func TestTryLoadFromValidUcmRoot(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte("ucm:\n  name: test\n"), 0o644))
	t.Setenv(ucm.RootEnv, dir)
	ctx = logdiag.InitContext(ctx)

	u := ucm.TryLoad(ctx)
	require.NotNil(t, u)
	assert.Equal(t, "test", u.Config.Ucm.Name)
	assert.False(t, logdiag.HasError(ctx))
}

func TestTryLoadReturnsNilWhenNoUcmYml(t *testing.T) {
	ctx := t.Context()
	// Defensively unset the env in case the host has it set.
	t.Setenv(ucm.RootEnv, "")
	require.NoError(t, os.Unsetenv(ucm.RootEnv))
	t.Chdir(t.TempDir())
	ctx = logdiag.InitContext(ctx)

	u := ucm.TryLoad(ctx)
	assert.Nil(t, u)
	assert.False(t, logdiag.HasError(ctx))
}
