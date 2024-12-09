package git

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
		out, err := FindLeafInTree(wd, ".git")
		assert.NoError(t, err)
		assert.Equal(t, root, out)
	}

	// Find from project root itself should work.
	{
		out, err := FindLeafInTree(root, ".git")
		assert.NoError(t, err)
		assert.Equal(t, root, out)
	}

	// Find for something that doesn't exist should work.
	{
		out, err := FindLeafInTree(root, "this-leaf-doesnt-exist-anywhere")
		assert.NoError(t, err)
		assert.Equal(t, "", out)
	}
}
