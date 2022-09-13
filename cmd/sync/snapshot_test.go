package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/utilities"
	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	root, err := git.Root()
	assert.NoError(t, err)
	projectDir := utilities.GetTestProject(t, root)

	f1, err := os.Create(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := os.Create(filepath.Join(projectDir, "world.txt"))
	assert.NoError(t, err)
	defer f2.Close()

	fileSet := git.NewFileSet(projectDir)
	files, err := fileSet.All()
	assert.NoError(t, err)
	state := snapshot{}
	change := state.diff(files)

	// New files are added to put
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 3)
	assert.Contains(t, change.put, "hello.txt")
	assert.Contains(t, change.put, "world.txt")
	assert.Contains(t, change.put, "databricks.yml")

	// Edited files are added to put
	_, err = f2.WriteString("I like clis")
	assert.NoError(t, err)
	files, err = fileSet.All()
	assert.NoError(t, err)
	change = state.diff(files)
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "world.txt")

	// Removed files are added to delete
	err = os.Remove(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	files, err = fileSet.All()
	assert.NoError(t, err)
	change = state.diff(files)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 0)
	assert.Contains(t, change.delete, "hello.txt")
}
