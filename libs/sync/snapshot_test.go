package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/testfile"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertKeysOfMap[T any](t *testing.T, m map[string]T, expectedKeys []string) {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	assert.ElementsMatch(t, expectedKeys, keys)
}

func TestDiff(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	f1 := testfile.CreateFile(t, filepath.Join(projectDir, "hello.txt"))
	defer f1.Close(t)
	worldFilePath := filepath.Join(projectDir, "world.txt")
	f2 := testfile.CreateFile(t, worldFilePath)
	defer f2.Close(t)

	// New files are put
	files, err := fileSet.Files()
	assert.NoError(t, err)
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Empty(t, change.delete)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "hello.txt")
	assert.Contains(t, change.put, "world.txt")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"hello.txt", "world.txt"})
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.RemoteToLocalNames)

	// world.txt is editted
	f2.Overwrite(t, "bunnies are cute.")
	assert.NoError(t, err)
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)

	assert.Empty(t, change.delete)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "world.txt")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"hello.txt", "world.txt"})
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"hello.txt": "hello.txt", "world.txt": "world.txt"}, state.RemoteToLocalNames)

	// hello.txt is deleted
	f1.Remove(t)
	assert.NoError(t, err)
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Empty(t, change.put)
	assert.Contains(t, change.delete, "hello.txt")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"world.txt"})
	assert.Equal(t, map[string]string{"world.txt": "world.txt"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"world.txt": "world.txt"}, state.RemoteToLocalNames)
}

func TestSymlinkDiff(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	err = os.Mkdir(filepath.Join(projectDir, "foo"), os.ModePerm)
	assert.NoError(t, err)

	f1 := testfile.CreateFile(t, filepath.Join(projectDir, "foo", "hello.txt"))
	defer f1.Close(t)

	err = os.Symlink(filepath.Join(projectDir, "foo"), filepath.Join(projectDir, "bar"))
	assert.NoError(t, err)

	files, err := fileSet.Files()
	assert.NoError(t, err)
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.put, 1)
}

func TestFolderDiff(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	err = os.Mkdir(filepath.Join(projectDir, "foo"), os.ModePerm)
	assert.NoError(t, err)
	f1 := testfile.CreateFile(t, filepath.Join(projectDir, "foo", "bar.py"))
	defer f1.Close(t)
	f1.Overwrite(t, "# Databricks notebook source\nprint(\"abc\")")

	files, err := fileSet.Files()
	assert.NoError(t, err)
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Empty(t, change.delete)
	assert.Empty(t, change.rmdir)
	assert.Len(t, change.mkdir, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.mkdir, "foo")
	assert.Contains(t, change.put, "foo/bar.py")

	f1.Remove(t)
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.rmdir, 1)
	assert.Empty(t, change.mkdir)
	assert.Empty(t, change.put)
	assert.Contains(t, change.delete, "foo/bar")
	assert.Contains(t, change.rmdir, "foo")
}

func TestPythonNotebookDiff(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	foo := testfile.CreateFile(t, filepath.Join(projectDir, "foo.py"))
	defer foo.Close(t)

	// Case 1: notebook foo.py is uploaded
	files, err := fileSet.Files()
	assert.NoError(t, err)
	foo.Overwrite(t, "# Databricks notebook source\nprint(\"abc\")")
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Empty(t, change.delete)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// Case 2: notebook foo.py is converted to python script by removing
	// magic keyword
	foo.Overwrite(t, "print(\"abc\")")
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.delete, "foo")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo.py"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo.py": "foo.py"}, state.RemoteToLocalNames)

	// Case 3: Python script foo.py is converted to a databricks notebook
	foo.Overwrite(t, "# Databricks notebook source\nprint(\"def\")")
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.delete, "foo.py")
	assertKeysOfMap(t, state.LastModifiedTimes, []string{"foo.py"})
	assert.Equal(t, map[string]string{"foo.py": "foo"}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo": "foo.py"}, state.RemoteToLocalNames)

	// Case 4: Python notebook foo.py is deleted, and its remote name is used in change.delete
	foo.Remove(t)
	assert.NoError(t, err)
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Empty(t, change.put)
	assert.Contains(t, change.delete, "foo")
	assert.Empty(t, state.LastModifiedTimes)
	assert.Equal(t, map[string]string{}, state.LocalToRemoteNames)
	assert.Equal(t, map[string]string{}, state.RemoteToLocalNames)
}

func TestErrorWhenIdenticalRemoteName(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	// upload should work since they point to different destinations
	pythonFoo := testfile.CreateFile(t, filepath.Join(projectDir, "foo.py"))
	defer pythonFoo.Close(t)
	vanillaFoo := testfile.CreateFile(t, filepath.Join(projectDir, "foo"))
	defer vanillaFoo.Close(t)
	files, err := fileSet.Files()
	assert.NoError(t, err)
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Empty(t, change.delete)
	assert.Len(t, change.put, 2)
	assert.Contains(t, change.put, "foo.py")
	assert.Contains(t, change.put, "foo")

	// errors out because they point to the same destination
	pythonFoo.Overwrite(t, "# Databricks notebook source\nprint(\"def\")")
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.ErrorContains(t, err, "both foo and foo.py point to the same remote file location foo. Please remove one of them from your local project")
}

func TestNoErrorRenameWithIdenticalRemoteName(t *testing.T) {
	ctx := context.Background()

	// Create temp project dir
	projectDir := t.TempDir()
	fileSet, err := git.NewFileSetAtRoot(vfs.MustNew(projectDir))
	require.NoError(t, err)
	state := Snapshot{
		SnapshotState: &SnapshotState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}

	// upload should work since they point to different destinations
	pythonFoo := testfile.CreateFile(t, filepath.Join(projectDir, "foo.py"))
	defer pythonFoo.Close(t)
	pythonFoo.Overwrite(t, "# Databricks notebook source\n")
	files, err := fileSet.Files()
	assert.NoError(t, err)
	change, err := state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Empty(t, change.delete)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.py")

	pythonFoo.Remove(t)
	sqlFoo := testfile.CreateFile(t, filepath.Join(projectDir, "foo.sql"))
	defer sqlFoo.Close(t)
	sqlFoo.Overwrite(t, "-- Databricks notebook source\n")
	files, err = fileSet.Files()
	assert.NoError(t, err)
	change, err = state.diff(ctx, files)
	assert.NoError(t, err)
	assert.Len(t, change.delete, 1)
	assert.Len(t, change.put, 1)
	assert.Contains(t, change.put, "foo.sql")
	assert.Contains(t, change.delete, "foo")
}

func defaultOptions(t *testing.T) *SyncOptions {
	return &SyncOptions{
		Host:             "www.foobar.com",
		RemotePath:       "/Repos/foo/bar",
		SnapshotBasePath: t.TempDir(),
	}
}

func TestNewSnapshotDefaults(t *testing.T) {
	opts := defaultOptions(t)
	snapshot, err := newSnapshot(context.Background(), opts)
	require.NoError(t, err)

	assert.Equal(t, LatestSnapshotVersion, snapshot.Version)
	assert.Equal(t, opts.RemotePath, snapshot.RemotePath)
	assert.Equal(t, opts.Host, snapshot.Host)
	assert.Empty(t, snapshot.LastModifiedTimes)
	assert.Empty(t, snapshot.RemoteToLocalNames)
	assert.Empty(t, snapshot.LocalToRemoteNames)
}

func TestOldSnapshotInvalidation(t *testing.T) {
	oldVersionSnapshot := `{
		"version": "v0",
		"host": "www.foobar.com",
		"remote_path": "/Repos/foo/bar",
		"last_modified_times": {},
		"local_to_remote_names": {},
		"remote_to_local_names": {}
	}`

	opts := defaultOptions(t)
	snapshotPath, err := SnapshotPath(opts)
	require.NoError(t, err)
	snapshotFile := testfile.CreateFile(t, snapshotPath)
	snapshotFile.Overwrite(t, oldVersionSnapshot)
	snapshotFile.Close(t)

	// assert snapshot did not get loaded
	snapshot, err := loadOrNewSnapshot(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, snapshot.New)
}

func TestNoVersionSnapshotInvalidation(t *testing.T) {
	noVersionSnapshot := `{
		"host": "www.foobar.com",
		"remote_path": "/Repos/foo/bar",
		"last_modified_times": {},
		"local_to_remote_names": {},
		"remote_to_local_names": {}
	}`

	opts := defaultOptions(t)
	snapshotPath, err := SnapshotPath(opts)
	require.NoError(t, err)
	snapshotFile := testfile.CreateFile(t, snapshotPath)
	snapshotFile.Overwrite(t, noVersionSnapshot)
	snapshotFile.Close(t)

	// assert snapshot did not get loaded
	snapshot, err := loadOrNewSnapshot(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, snapshot.New)
}

func TestLatestVersionSnapshotGetsLoaded(t *testing.T) {
	latestVersionSnapshot := fmt.Sprintf(`{
			"version": "%s",
			"host": "www.foobar.com",
			"remote_path": "/Repos/foo/bar",
			"last_modified_times": {},
			"local_to_remote_names": {},
			"remote_to_local_names": {}
	}`, LatestSnapshotVersion)

	opts := defaultOptions(t)
	snapshotPath, err := SnapshotPath(opts)
	require.NoError(t, err)
	snapshotFile := testfile.CreateFile(t, snapshotPath)
	snapshotFile.Overwrite(t, latestVersionSnapshot)
	snapshotFile.Close(t)

	// assert snapshot gets loaded
	snapshot, err := loadOrNewSnapshot(context.Background(), opts)
	require.NoError(t, err)
	assert.False(t, snapshot.New)
	assert.Equal(t, LatestSnapshotVersion, snapshot.Version)
	assert.Equal(t, "www.foobar.com", snapshot.Host)
	assert.Equal(t, "/Repos/foo/bar", snapshot.RemotePath)
}
