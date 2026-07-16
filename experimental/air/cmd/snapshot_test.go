package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRootPath(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj"), 0o755))

	// root_path "." resolves against configDir to an absolute path whose basename is
	// the real directory name — not ".".
	got, err := resolveRootPath(ctx, ".", filepath.Join(dir, "proj"))
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
	assert.Equal(t, "proj", filepath.Base(got))

	// A relative subpath resolves against configDir and keeps its own basename.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj", "sub"), 0o755))
	got, err = resolveRootPath(ctx, "sub", filepath.Join(dir, "proj"))
	require.NoError(t, err)
	assert.Equal(t, "sub", filepath.Base(got))

	// A non-existent path errors.
	_, err = resolveRootPath(ctx, "missing", dir)
	require.Error(t, err)
}
