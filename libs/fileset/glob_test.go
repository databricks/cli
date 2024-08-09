package fileset

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/require"
)

func collectRelativePaths(files []File) []string {
	relativePaths := make([]string, 0)
	for _, f := range files {
		relativePaths = append(relativePaths, f.Relative)
	}
	return relativePaths
}

func TestGlobFileset(t *testing.T) {
	root := vfs.MustNew("../filer")
	entries, err := root.ReadDir(".")
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)

	// +1 as there's one folder in ../filer
	require.Equal(t, len(files), len(entries)+1)
	for _, f := range files {
		exists := slices.ContainsFunc(entries, func(de fs.DirEntry) bool {
			return de.Name() == path.Base(f.Relative)
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
	root := vfs.MustNew("../filer")
	entries, err := root.ReadDir(".")
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	// +1 as there's one folder in ../filer
	require.Equal(t, len(files), len(entries)+1)
}

func TestGlobFilesetRecursively(t *testing.T) {
	root := vfs.MustNew("../git")
	entries := make([]string, 0)
	err := fs.WalkDir(root, "testdata", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"testdata/*",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}

func TestGlobFilesetDir(t *testing.T) {
	root := vfs.MustNew("../git")
	entries := make([]string, 0)
	err := fs.WalkDir(root, "testdata/a", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"testdata/a",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}

func TestGlobFilesetDoubleQuotesWithFilePatterns(t *testing.T) {
	root := vfs.MustNew("../git")
	entries := make([]string, 0)
	err := fs.WalkDir(root, "testdata", func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".txt") {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(root, []string{
		"testdata/**/*.txt",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}
