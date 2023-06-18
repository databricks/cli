package filer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalClientAtRootOfFilesystem(t *testing.T) {
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
