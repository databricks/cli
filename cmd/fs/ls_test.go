package fs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsoluteLsBaseDir(t *testing.T) {
	t.Run("relative local", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		got, err := absoluteLsBaseDir("child", true)
		require.NoError(t, err)
		assert.Equal(t, filepath.ToSlash(dir)+"/child", got)
	})

	t.Run("dbfs preserves scheme and path", func(t *testing.T) {
		got, err := absoluteLsBaseDir("dbfs:/parent", true)
		require.NoError(t, err)
		assert.Equal(t, "dbfs:/parent", got)
	})

	t.Run("relative output remains relative", func(t *testing.T) {
		got, err := absoluteLsBaseDir("child", false)
		require.NoError(t, err)
		assert.Equal(t, "child", got)
	})
}
