package fileset

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
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
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(cwd, "..", "filer")

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	g, err := NewGlobSet(vfs.MustNew(root), []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)

	require.Equal(t, len(files), len(entries))
	for _, f := range files {
		exists := slices.ContainsFunc(entries, func(de fs.DirEntry) bool {
			return de.Name() == path.Base(f.Relative)
		})
		require.True(t, exists)
	}

	g, err = NewGlobSet(vfs.MustNew(root), []string{
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

	g, err := NewGlobSet(vfs.MustNew(root), []string{
		"./*.go",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)

	require.Equal(t, len(files), len(entries))
}

func TestGlobFilesetRecursively(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(cwd, "..", "git")

	entries := make([]string, 0)
	err = fs.WalkDir(os.DirFS(filepath.Join(root)), "testdata", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(vfs.MustNew(root), []string{
		"testdata/*",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}

func TestGlobFilesetDir(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(cwd, "..", "git")

	entries := make([]string, 0)
	err = fs.WalkDir(os.DirFS(filepath.Join(root)), "testdata/a", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(vfs.MustNew(root), []string{
		"testdata/a",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}

func TestGlobFilesetDoubleQuotesWithFilePatterns(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(cwd, "..", "git")

	entries := make([]string, 0)
	err = fs.WalkDir(os.DirFS(filepath.Join(root)), "testdata", func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".txt") {
			entries = append(entries, path)
		}
		return nil
	})
	require.NoError(t, err)

	g, err := NewGlobSet(vfs.MustNew(root), []string{
		"testdata/**/*.txt",
	})
	require.NoError(t, err)

	files, err := g.All()
	require.NoError(t, err)
	require.ElementsMatch(t, entries, collectRelativePaths(files))
}
