package sync

import (
	"os"
	"path/filepath"
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
	change, err = state.diff(files)
	assert.NoError(t, err)

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

// func TestFilesWithSameRemoteNameNotAllowed(t *testing.T) {
// 	// Create temp project dir
// 	projectDir := t.TempDir()

// 	// Create notebook
// 	fooPath := filepath.Join(projectDir, "foo.py")
// 	f, err := os.Create(fooPath)
// 	assert.NoError(t, err)
// 	defer f.Close()

// 	// Create vanilla foo file
// 	foo2Path := filepath.Join(projectDir, "foo")
// 	f2, err := os.Create(foo2Path)
// 	assert.NoError(t, err)
// 	defer f2.Close()

// 	f.Write([]byte("# Databricks notebook source\nprint(\"abc\")"))
// 	fileSet := git.NewFileSet(projectDir)
// 	files, err := fileSet.All()
// 	assert.NoError(t, err)
// 	state := Snapshot{
// 		LastUpdatedTimes:   make(map[string]time.Time),
// 		LocalToRemoteNames: make(map[string]string),
// 		RemoteToLocalNames: make(map[string]string),
// 	}
// 	change, err := state.diff(files)
// 	assert.NoError(t, err)
// }

// Does writing update mtime?
// func TestDebug(t *testing.T) {
// 	// Create temp project dir
// 	projectDir := t.TempDir()

// 	fooPath := filepath.Join(projectDir, "foo")
// 	f, err := os.Create(fooPath)
// 	assert.NoError(t, err)

// 	f.WriteString()

// }

func getFDOffset(t *testing.T, f *os.File) int64 {
	offset, err := f.Seek(0, 1)
	assert.NoError(t, err)
	return offset
}

func TestPythonNotebookDiff(t *testing.T) {
	assert.True(t, false)
	// Create temp project dir
	projectDir := t.TempDir()

	// Create notebook
	fooPath := filepath.Join(projectDir, "foo.py")
	f, err := os.Create(fooPath)
	assert.NoError(t, err)
	defer f.Close()

	t.Logf("[AAAA] f offset before write: %v", getFDOffset(t, f))
	f.Write([]byte("# Databricks notebook source\nprint(\"abc\")"))
	t.Logf("[AAAA] f offset after write: %v", getFDOffset(t, f))
	fileSet := git.NewFileSet(projectDir)
	files, err := fileSet.All()
	assert.NoError(t, err)
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	// Case 1: notebook foo.py is uploaded
	change, err := state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// Case 2: notebook foo.py is converted to python script by removing
	// magic keyword
	// We manually update mtime after write because github file system does not
	t.Logf("[AAAA] f offset before truncate: %v", getFDOffset(t, f))
	err = os.Truncate(fooPath, 0)
	t.Logf("[AAAA] f offset after truncate: %v", getFDOffset(t, f))
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

	// assert.Eventually(t, func() bool {
	// 	content, err := os.ReadFile(fooPath)
	// 	assert.NoError(t, err)
	// 	return strings.Contains(string(content), "# Databricks notebook source")
	// }, 3*time.Second, time.Second)

	content, err := os.ReadFile(fooPath)
	assert.NoError(t, err)
	t.Logf("[CASE 3] before foo contents: %s", content)
	t.Logf("[CASE 3] before state: %+v", state)
	t.Logf("[CASE 3] fooPath: %s", fooPath)

	// Case 3: Python script foo.py is converted to a databricks notebook
	// by adding magic keyword
	t.Logf("[AAAA] f offset before write2: %v", getFDOffset(t, f))
	_, err = f.Seek(0, 0)
	assert.NoError(t, err)
	t.Logf("[AAAA] f offset after seek reset: %v", getFDOffset(t, f))
	f.Write([]byte("# Databricks notebook source\nprint(\"def\")"))
	t.Logf("[AAAA] f offset after write2: %v", getFDOffset(t, f))

	fooInfo, err = os.Stat(fooPath)
	assert.NoError(t, err)
	os.Chtimes(fooPath,
		fooInfo.ModTime().Add(time.Minute),
		fooInfo.ModTime().Add(time.Minute))

	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)

	content, err = os.ReadFile(fooPath)
	assert.NoError(t, err)
	t.Logf("[CASE 3] after foo contents: %s", content)
	t.Logf("[CASE 3] after state: %+v", state)

	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.delete, "foo.py")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// Case 4: Python notebook foo.py is deleted, and its remote name is used in change.delete
	err = os.Remove(fooPath)
	assert.NoError(t, err)
	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 0)
	assert.Contains(t, change.delete, "foo")
	assert.Len(t, state.LastUpdatedTimes, 0)
	assert.Equal(t, map[string]string{}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{}, state.RemoteToLocalNames)
}
