package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStateDirReturnsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	u := &ucm.Ucm{RootPath: root}
	u.Config.Ucm.Target = "dev"

	dir, err := localStateDir(u, "tmp")
	require.NoError(t, err)

	want := filepath.Join(root, ".databricks", "ucm", "dev", "tmp")
	assert.Equal(t, want, dir)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLocalStateDirErrorsWhenTargetUnset(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir()}
	_, err := localStateDir(u, "tmp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target")
}

func TestLocalStateDirErrorsWhenUcmNil(t *testing.T) {
	_, err := localStateDir(nil, "tmp")
	require.Error(t, err)
}
