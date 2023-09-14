package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFileSetAll(t *testing.T, path string) {
	fileSet, err := NewFileSet(path)
	require.NoError(t, err)
	files, err := fileSet.All()
	require.NoError(t, err)
	require.Len(t, files, 3)
	assert.Equal(t, filepath.Join("a", "b", "world.txt"), files[0].Relative)
	assert.Equal(t, filepath.Join("a", "hello.txt"), files[1].Relative)
	assert.Equal(t, filepath.Join("databricks.yml"), files[2].Relative)
}

func TestFileSetListAllInRepo(t *testing.T) {
	testFileSetAll(t, "./testdata")
}

func TestFileSetListAllInTempDir(t *testing.T) {
	testFileSetAll(t, copyTestdata(t, "./testdata"))
}

func TestFileSetNonCleanRoot(t *testing.T) {
	// Test what happens if the root directory can be simplified.
	// Path simplification is done by most filepath functions.
	// This should yield the same result as above test.
	fileSet, err := NewFileSet("./testdata/../testdata")
	require.NoError(t, err)
	files, err := fileSet.All()
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestFileSetAddsCacheDirToGitIgnore(t *testing.T) {
	projectDir := t.TempDir()
	fileSet, err := NewFileSet(projectDir)
	require.NoError(t, err)
	fileSet.EnsureValidGitIgnoreExists()

	gitIgnorePath := filepath.Join(projectDir, ".gitignore")
	assert.FileExists(t, gitIgnorePath)
	fileBytes, err := os.ReadFile(gitIgnorePath)
	assert.NoError(t, err)
	assert.Contains(t, string(fileBytes), ".databricks")
}

func TestFileSetDoesNotCacheDirToGitIgnoreIfAlreadyPresent(t *testing.T) {
	projectDir := t.TempDir()
	gitIgnorePath := filepath.Join(projectDir, ".gitignore")

	fileSet, err := NewFileSet(projectDir)
	require.NoError(t, err)
	err = os.WriteFile(gitIgnorePath, []byte(".databricks"), 0o644)
	require.NoError(t, err)

	fileSet.EnsureValidGitIgnoreExists()

	b, err := os.ReadFile(gitIgnorePath)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(b), ".databricks"))
}
