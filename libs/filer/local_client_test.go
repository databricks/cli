package filer

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalClientAtRootForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("this test is only meant for windows")
	}
	tmpDir := t.TempDir()
	ctx := context.Background()

	// create filer root at "root"
	f, err := NewLocalClient("/")
	require.NoError(t, err)

	// Assert stat call to the temp dir succeeds.
	info, err := f.Stat(ctx, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Base(tmpDir), info.Name())
	assert.True(t, info.IsDir())
}
