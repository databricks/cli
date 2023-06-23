package fs

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilerForPathForLocalPaths(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	f, path, err := FilerForPath(ctx, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, tmpDir, path)

	info, err := f.Stat(ctx, path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFilerForPathForInvalidScheme(t *testing.T) {
	ctx := context.Background()

	_, _, err := FilerForPath(ctx, "dbf:/a")
	assert.ErrorContains(t, err, "invalid scheme")

	_, _, err = FilerForPath(ctx, "foo:a")
	assert.ErrorContains(t, err, "invalid scheme")

	_, _, err = FilerForPath(ctx, "file:/a")
	assert.ErrorContains(t, err, "invalid scheme")
}

func testWindowsFilerForPath(t *testing.T, ctx context.Context, fullPath string) {
	f, path, err := FilerForPath(ctx, fullPath)
	assert.NoError(t, err)

	// Assert path remains unchanged
	assert.Equal(t, path, fullPath)

	// Assert local client is created
	_, ok := f.(*filer.LocalClient)
	assert.True(t, ok)
}

func TestFilerForWindowsLocalPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	ctx := context.Background()
	testWindowsFilerForPath(t, ctx, `c:\abc`)
	testWindowsFilerForPath(t, ctx, `c:abc`)
	testWindowsFilerForPath(t, ctx, `d:\abc`)
	testWindowsFilerForPath(t, ctx, `d:\abc`)
	testWindowsFilerForPath(t, ctx, `f:\abc\ef`)
}
