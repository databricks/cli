package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/bricks/git"
	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()

	f1, err := os.Create(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	defer f1.Close()
	worldFilePath := filepath.Join(projectDir, "world.txt")
	f2, err := os.Create(worldFilePath)
	assert.NoError(t, err)
	defer f2.Close()

	fileSet := git.NewFileSet(projectDir)
	files, err := fileSet.All()
	assert.NoError(t, err)
	state := snapshot{
		LastModifiedTimes: make(map[string]time.Time),
	}
	change := state.diff(files)

	// New files are added to put
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "hello.txt")
	assert.Contains(t, change.put, "world.txt")

	// Edited files are added to put.
	// File system in the github actions env does not update
	// mtime on writes to a file. So we are manually editting it
	// instead of writing to world.txt
	worldFileInfo, err := os.Stat(worldFilePath)
	assert.NoError(t, err)
	os.Chtimes(worldFilePath,
		worldFileInfo.ModTime().Add(time.Nanosecond),
		worldFileInfo.ModTime().Add(time.Nanosecond))

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
