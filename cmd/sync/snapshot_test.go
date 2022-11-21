package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/bricks/git"
	"github.com/stretchr/testify/assert"
)

func assertKeysOfMap(t *testing.T, m map[string]time.Time, expectedKeys []string) {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	assert.ElementsMatch(t, expectedKeys, keys)
}

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
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}
	change, err := state.diff(files)
	assert.NoError(t, err)

	// New files are added to put
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "hello.txt")
	assert.Contains(t, change.put, "world.txt")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"hello.txt", "world.txt"})
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.RemoteToLocalNames)

	t.Logf("[AAAA] state %+v: ", state)
	t.Logf("[AAAA] files %+v: ", files)
	t.Logf("[AAAA] files[0].Modified() %+v: ", files[0].Modified())
	helloInfo, err := os.Stat(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	t.Logf("[AAAA] helloInfo.ModTime() %+v: ", helloInfo.ModTime())
	worldInfo, err := os.Stat(filepath.Join(projectDir, "world.txt"))
	assert.NoError(t, err)
	t.Logf("[AAAA] worldInfo.ModTime() %+v: ", worldInfo.ModTime())

	// Edited files are added to put.
	// File system in the github actions env does not update
	// mtime on writes to a file. So we are manually editting it
	// instead of writing to world.txt
	worldFileInfo, err := os.Stat(worldFilePath)
	assert.NoError(t, err)
	os.Chtimes(worldFilePath,
		worldFileInfo.ModTime().Add(time.Minute),
		worldFileInfo.ModTime().Add(time.Minute))

	assert.NoError(t, err)
	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)

	t.Logf("[AAAA] state %+v: ", state)
	t.Logf("[AAAA] files %+v: ", files)
	t.Logf("[AAAA] files[0].Modified() %+v: ", files[0].Modified())
	helloInfo, err = os.Stat(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	t.Logf("[AAAA] helloInfo.ModTime() %+v: ", helloInfo.ModTime())
	worldInfo, err = os.Stat(filepath.Join(projectDir, "world.txt"))
	assert.NoError(t, err)
	t.Logf("[AAAA] worldInfo.ModTime() %+v: ", worldInfo.ModTime())

	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "world.txt")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"hello.txt", "world.txt"})
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.RemoteToLocalNames)

	// Removed files are added to delete
	err = os.Remove(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 0)
	assert.Contains(t, change.delete, "hello.txt")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"world.txt"})
	assert.Equal(t, map[string]string{"world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"world.txt": "world.txt"}, state.RemoteToLocalNames)
}

func TestIncrementTimestamp(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()

	f1, err := os.Create(filepath.Join(projectDir, "hello.txt"))
	assert.NoError(t, err)
	defer f1.Close()

}

// TODO: create fooPath variable
func TestPythonNotebookDiff(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()

	fooPath := filepath.Join(projectDir, "foo.py")
	f, err := os.Create(fooPath)
	assert.NoError(t, err)
	defer f.Close()

	f.Write([]byte("# Databricks notebook source\nprint(\"abc\")"))

	fileSet := git.NewFileSet(projectDir)
	files, err := fileSet.All()
	assert.NoError(t, err)
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}
	change, err := state.diff(files)
	assert.NoError(t, err)

	// notebook is uploaded with its local name
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// convert notebook -> python script
	// File system in the github actions env does not update
	// mtime on writes to a file. So we are manually editting it
	err = os.Truncate(fooPath, 0)
	assert.NoError(t, err)
	fooInfo, err := os.Stat(fooPath)
	assert.NoError(t, err)
	os.Chtimes(fooPath,
		fooInfo.ModTime().Add(time.Minute),
		fooInfo.ModTime().Add(time.Minute))

	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)

	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.delete, "foo")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo.py"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo.py": "foo.py"}, state.RemoteToLocalNames)

	// convert python script -> notebook
	// File system in the github actions env does not update
	// mtime on writes to a file. So we are manually editting it
	f2, err := os.Open(fooPath)
	assert.NoError(t, err)
	defer f2.Close()
	f.Write([]byte("# Databricks notebook source\nprint(\"def\")"))
	os.Chtimes(fooPath,
		fooInfo.ModTime().Add(time.Minute),
		fooInfo.ModTime().Add(time.Minute))

	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)

	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.delete, "foo.py")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// // Removed notebook are added to delete with remote name
	// err = os.Remove(filePath)
	// assert.NoError(t, err)
	// files, err = fileSet.All()
	// assert.NoError(t, err)
	// change, err = state.diff(files)
	// assert.NoError(t, err)
	// assert.Len(t, change.delete, 1)
	// assert.Len(t, change.put, 0)
	// assert.Contains(t, change.delete, "foo")
	// assert.Len(t, state.LastUpdatedTimes, 0)
	// assert.Equal(t, map[string]string{}, state.LocalToRemoteNames)
	// assert.Equal(t, map[string]string{}, state.RemoteToLocalNames)
}
