package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/bricks/utilities"
	"github.com/stretchr/testify/assert"
)

func TestRecusiveListFile(t *testing.T) {
	root, err := Root()
	assert.NoError(t, err)
	projectName := "test-fileset-recursive-list"
	projectDir := filepath.Join(root, "tmp", projectName)
	utilities.CreateTestProject(t, root, projectName)
	defer utilities.DeleteTestProject(t, root, projectName)
	f3, err := os.Create(filepath.Join(projectDir, ".gitignore"))
	assert.NoError(t, err)
	defer f3.Close()
	f3.WriteString(".gitignore\nd")

	// Check the config file is being tracked
	fileSet := NewFileSet(projectDir)
	files, err := fileSet.RecursiveListTrackedFiles(projectDir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "databricks.yml", files[0].Relative)

	// Check that newly added files not in .gitignore
	// are being tracked
	dir1 := filepath.Join(projectDir, "a", "b", "c")
	dir2 := filepath.Join(projectDir, "d", "e")
	err = os.MkdirAll(dir2, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(dir1, 0o755)
	assert.NoError(t, err)
	f1, err := os.Create(filepath.Join(projectDir, "a/b/c/hello.txt"))
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := os.Create(filepath.Join(projectDir, "d/e/world.txt"))
	assert.NoError(t, err)
	defer f2.Close()
	assert.NoError(t, err)

	files, err = fileSet.RecursiveListTrackedFiles(projectDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "databricks.yml", files[0].Relative)
	assert.Equal(t, "a/b/c/hello.txt", files[1].Relative)
}
