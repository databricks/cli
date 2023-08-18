package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/require"
)

func createFile(dir string, name string) error {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}

	return f.Close()
}

func setupFiles(t *testing.T) string {
	dir := t.TempDir()

	err := createFile(dir, "a.go")
	require.NoError(t, err)

	err = createFile(dir, "b.go")
	require.NoError(t, err)

	err = createFile(dir, "ab.go")
	require.NoError(t, err)

	err = createFile(dir, "abc.go")
	require.NoError(t, err)

	err = createFile(dir, "c.go")
	require.NoError(t, err)

	err = createFile(dir, "d.go")
	require.NoError(t, err)

	dbDir := filepath.Join(dir, ".databricks")
	err = os.Mkdir(dbDir, 0755)
	require.NoError(t, err)

	err = createFile(dbDir, "e.go")
	require.NoError(t, err)

	return dir

}

func TestGetFileSet(t *testing.T) {
	ctx := context.Background()

	dir := setupFiles(t)
	fileSet, err := git.NewFileSet(dir)
	require.NoError(t, err)

	err = fileSet.EnsureValidGitIgnoreExists()
	require.NoError(t, err)

	inc, err := fileset.NewGlobSet(dir, []string{})
	require.NoError(t, err)

	excl, err := fileset.NewGlobSet(dir, []string{})
	require.NoError(t, err)

	s := &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err := getFileList(ctx, s)
	require.NoError(t, err)
	require.Equal(t, len(fileList), 7)

	inc, err = fileset.NewGlobSet(dir, []string{})
	require.NoError(t, err)

	excl, err = fileset.NewGlobSet(dir, []string{"*.go"})
	require.NoError(t, err)

	s = &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err = getFileList(ctx, s)
	require.NoError(t, err)
	require.Equal(t, len(fileList), 1)

	inc, err = fileset.NewGlobSet(dir, []string{".databricks/*.*"})
	require.NoError(t, err)

	excl, err = fileset.NewGlobSet(dir, []string{})
	require.NoError(t, err)

	s = &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err = getFileList(ctx, s)
	require.NoError(t, err)
	require.Equal(t, len(fileList), 8)

}
