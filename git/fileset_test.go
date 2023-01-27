package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSetRecursiveListFiles(t *testing.T) {
	fileSet, err := NewFileSet("./testdata")
	require.NoError(t, err)
	files, err := fileSet.RecursiveListFiles("./testdata")
	require.NoError(t, err)
	require.Len(t, files, 6)
	assert.Equal(t, filepath.Join(".gitignore"), files[0].Relative)
	assert.Equal(t, filepath.Join("a", ".gitignore"), files[1].Relative)
	assert.Equal(t, filepath.Join("a", "b", ".gitignore"), files[2].Relative)
	assert.Equal(t, filepath.Join("a", "b", "world.txt"), files[3].Relative)
	assert.Equal(t, filepath.Join("a", "hello.txt"), files[4].Relative)
	assert.Equal(t, filepath.Join("databricks.yml"), files[5].Relative)
}

func TestFileSetNonCleanRoot(t *testing.T) {
	// Test what happens if the root directory can be simplified.
	// Path simplification is done by most filepath functions.
	// This should yield the same result as above test.
	fileSet, err := NewFileSet("./testdata/../testdata")
	require.NoError(t, err)
	files, err := fileSet.All()
	require.NoError(t, err)
	assert.Len(t, files, 6)
}
