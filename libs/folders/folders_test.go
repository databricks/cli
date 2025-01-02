package folders

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDirWithLeaf(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	root := filepath.Join(wd, "..", "..")

	// Find from working directory should work.
	{
		out, err := FindDirWithLeaf(wd, ".git")
		assert.NoError(t, err)
		assert.Equal(t, out, root)
	}

	// Find from project root itself should work.
	{
		out, err := FindDirWithLeaf(root, ".git")
		assert.NoError(t, err)
		assert.Equal(t, out, root)
	}

	// Find for something that doesn't exist should work.
	{
		out, err := FindDirWithLeaf(root, "this-leaf-doesnt-exist-anywhere")
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Equal(t, "", out)
	}
}
