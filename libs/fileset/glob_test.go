package fileset

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobFileset(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(cwd, "..", "filer")

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)

	require.Equal(t, len(files), len(entries))
	for _, f := range files {
		exists := slices.ContainsFunc(entries, func(de fs.DirEntry) bool {
			return de.Name() == f.Name()
		})
		require.True(t, exists)
	}

	g, err = NewGlobSet(root, []string{
		"./*.js",
	})
	require.NoError(t, err)

	files, err = g.All()
	require.NoError(t, err)
	require.Equal(t, len(files), 0)
}

func TestGlobFilesetWithRelativeRoot(t *testing.T) {
	root := filepath.Join("..", "filer")

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)

	require.Equal(t, len(files), len(entries))
	for _, f := range files {
		require.True(t, filepath.IsAbs(f.Absolute))
	}
}
