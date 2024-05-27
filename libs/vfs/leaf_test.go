package vfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLeafInTree(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	root := filepath.Join(wd, "..", "..")

	// Find from working directory should work.
	{
		out, err := FindLeafInTree(MustNew(wd), ".git")
		assert.NoError(t, err)
		assert.Equal(t, root, out.Native())
	}

	// Find from project root itself should work.
	{
		out, err := FindLeafInTree(MustNew(root), ".git")
		assert.NoError(t, err)
		assert.Equal(t, root, out.Native())
	}

	// Find for something that doesn't exist should work.
	{
		out, err := FindLeafInTree(MustNew(root), "this-leaf-doesnt-exist-anywhere")
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Equal(t, nil, out)
	}
}
