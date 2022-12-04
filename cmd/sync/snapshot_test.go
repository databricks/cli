package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
	"github.com/stretchr/testify/assert"
)

type testFile struct {
	mtime time.Time
	fd    *os.File
	path  string
	// to make close idempotent
	isOpen bool
}

func createFile(t *testing.T, path string) *testFile {
	f, err := os.Create(path)
	assert.NoError(t, err)

	fileInfo, err := os.Stat(path)
	assert.NoError(t, err)

	return &testFile{
		path:   path,
		fd:     f,
		mtime:  fileInfo.ModTime(),
		isOpen: true,
	}
}

func (f *testFile) close(t *testing.T) {
	if f.isOpen {
		err := f.fd.Close()
		assert.NoError(t, err)
		f.isOpen = false
	}
}

func (f *testFile) overwrite(t *testing.T, s string) {
	err := os.Truncate(f.path, 0)
	assert.NoError(t, err)

	_, err = f.fd.Seek(0, 0)
	assert.NoError(t, err)

	_, err = f.fd.WriteString(s)
	assert.NoError(t, err)

	// We manually update mtime after write because github actions file
	// system does not :')
	err = os.Chtimes(f.path, f.mtime.Add(time.Minute), f.mtime.Add(time.Minute))
	assert.NoError(t, err)
	f.mtime = f.mtime.Add(time.Minute)
}

func (f *testFile) remove(t *testing.T) {
	f.close(t)
	err := os.Remove(f.path)
	assert.NoError(t, err)
}

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
	fileSet := git.NewFileSet(projectDir)
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	f1 := createFile(t, filepath.Join(projectDir, "hello.txt"))
	defer f1.close(t)
	worldFilePath := filepath.Join(projectDir, "world.txt")
	f2 := createFile(t, worldFilePath)
	defer f2.close(t)

	// New files are put
	files, err := fileSet.All()
	assert.NoError(t, err)
	change, err := state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "hello.txt")
	assert.Contains(t, change.put, "world.txt")
	assertKeysOfMap(t, state.LastUpdatedTimes, []string{"hello.txt", "world.txt"})
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.RemoteToLocalNames)

	// world.txt is editted
	f2.overwrite(t, "bunnies are cute.")
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

	// hello.txt is deleted
	f1.remove(t)
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

func TestPythonNotebookDiff(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()
	fileSet := git.NewFileSet(projectDir)
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	foo := createFile(t, filepath.Join(projectDir, "foo.py"))
	defer foo.close(t)

	// Case 1: notebook foo.py is uploaded
	files, err := fileSet.All()
	assert.NoError(t, err)
	foo.overwrite(t, "# Databricks notebook source\nprint(\"abc\")")
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
	foo.overwrite(t, "print(\"abc\")")
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

	// Case 3: Python script foo.py is converted to a databricks notebook
	foo.overwrite(t, "# Databricks notebook source\nprint(\"def\")")
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

	// Case 4: Python notebook foo.py is deleted, and its remote name is used in change.delete
	foo.remove(t)
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

func TestErrorWhenIdenticalRemoteName(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()
	fileSet := git.NewFileSet(projectDir)
	state := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	// upload should work since they point to different destinations
	pythonFoo := createFile(t, filepath.Join(projectDir, "foo.py"))
	defer pythonFoo.close(t)
	vanillaFoo := createFile(t, filepath.Join(projectDir, "foo"))
	defer vanillaFoo.close(t)
	files, err := fileSet.All()
	assert.NoError(t, err)
	change, err := state.diff(files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 0)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.put, "foo")

	// errors out because they point to the same destination
	pythonFoo.overwrite(t, "# Databricks notebook source\nprint(\"def\")")
	files, err = fileSet.All()
	assert.NoError(t, err)
	change, err = state.diff(files)
	assert.ErrorContains(t, err, "both foo and foo.py point to the same remote file location foo. Please remove one of them from your local project")
}

func TestNewSnapshotDefaults(t *testing.T) {
	// Create temp project dir
	projectDir := t.TempDir()
	ctx := context.TODO()
	err := os.Setenv("DATABRICKS_HOST", "www.foobar.com")
	assert.NoError(t, err)
	ctx, err = project.Initialize(ctx, projectDir, "development")
	assert.NoError(t, err)
	snapshot, err := newSnapshot(ctx, "/Repos/foo/bar")
	prj := project.Get(ctx)
	assert.NoError(t, err)

	assert.Equal(t, LatestSnapshotVersion, snapshot.Version)
	assert.Equal(t, "/Repos/foo/bar", snapshot.RemotePath)
	assert.Equal(t, prj.WorkspacesClient().Config.Host, snapshot.Host)
	assert.Empty(t, snapshot.LastUpdatedTimes)
	assert.Empty(t, snapshot.RemoteToLocalNames)
	assert.Empty(t, snapshot.LocalToRemoteNames)
}

func TestSnapshotVersionInvalidation(t *testing.T) {
	oldVersionSnapshot := `{
			"version": "v0",
			"host": "www.foobar.com",
			"remote_path": "/Repos/foo/bar",
			"last_modified_times": {},
			"local_to_remote_names": {},
			"remote_to_local_names": {}
	}`
	noVersionSnapshot := `{
			"host": "www.foobar.com",
			"remote_path": "/Repos/foo/bar",
			"last_modified_times": {},
			"local_to_remote_names": {},
			"remote_to_local_names": {}
	}`
	latestVersionSnapshot := fmt.Sprintf(`{
			"version": "%s",
			"host": "www.foobar.com",
			"remote_path": "/Repos/foo/bar",
			"last_modified_times": {},
			"local_to_remote_names": {},
			"remote_to_local_names": {}
	}`, LatestSnapshotVersion)

	// Create temp project dir
	projectDir := t.TempDir()
	ctx := context.TODO()
	err := os.Setenv("DATABRICKS_HOST", "www.foobar.com")
	assert.NoError(t, err)
	ctx, err = project.Initialize(ctx, projectDir, "development")
	assert.NoError(t, err)
	emptySnapshot := Snapshot{
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}
	snapshotPath, err := emptySnapshot.getPath(ctx)
	assert.NoError(t, err)

	// case: old version snapshot
	err = os.WriteFile(snapshotPath, []byte(oldVersionSnapshot), os.ModePerm)
	assert.NoError(t, err)
	assert.FileExists(t, snapshotPath)
	snapshot := emptySnapshot
	err = snapshot.loadSnapshot(ctx)
	assert.NoError(t, err)
	// snapshot invalidated
	assert.NoFileExists(t, snapshotPath)
	assert.Equal(t, emptySnapshot, snapshot)

	// case: no version snapshot
	err = os.WriteFile(snapshotPath, []byte(noVersionSnapshot), os.ModePerm)
	assert.NoError(t, err)
	assert.FileExists(t, snapshotPath)
	snapshot = emptySnapshot
	err = snapshot.loadSnapshot(ctx)
	assert.NoError(t, err)
	// snapshot invalidated
	assert.NoFileExists(t, snapshotPath)
	assert.Equal(t, emptySnapshot, snapshot)

	// case: latest version snapshot
	err = os.WriteFile(snapshotPath, []byte(latestVersionSnapshot), os.ModePerm)
	assert.NoError(t, err)
	assert.FileExists(t, snapshotPath)
	err = snapshot.loadSnapshot(ctx)
	// snapshot not invalidated
	assert.FileExists(t, snapshotPath)
	assert.NoError(t, err)
	assert.NotEqual(t, emptySnapshot, snapshot)
}
