package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirIsEmpty(t *testing.T) {
	empty := t.TempDir()
	assert.True(t, dirIsEmpty(empty))

	withFile := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(withFile, "f"), []byte("x"), 0o644))
	assert.False(t, dirIsEmpty(withFile))

	assert.False(t, dirIsEmpty(filepath.Join(empty, "does-not-exist")))
}

func TestDefaultRepoName(t *testing.T) {
	t.Run("valid dir name is used", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "my-app")
		require.NoError(t, os.Mkdir(dir, 0o755))
		t.Chdir(dir)
		assert.Equal(t, "my-app", defaultRepoName())
	})

	t.Run("invalid dir name falls back", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "Bad_Name")
		require.NoError(t, os.Mkdir(dir, 0o755))
		t.Chdir(dir)
		assert.Equal(t, "my-app", defaultRepoName())
	})
}
