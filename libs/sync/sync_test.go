package sync

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/require"
)

func setupFiles(t *testing.T) string {
	dir := t.TempDir()

	for _, f := range []([]string){
		[]string{dir, "a.go"},
		[]string{dir, "b.go"},
		[]string{dir, "ab.go"},
		[]string{dir, "abc.go"},
		[]string{dir, "c.go"},
		[]string{dir, "d.go"},
		[]string{dir, ".databricks", "e.go"},
		[]string{dir, "test", "sub1", "f.go"},
		[]string{dir, "test", "sub1", "sub2", "g.go"},
		[]string{dir, "test", "sub1", "sub2", "h.txt"},
	} {
		testutil.Touch(t, f...)
	}

	return dir
}

func TestGetFileSet(t *testing.T) {
	ctx := context.Background()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)
	fileSet, err := git.NewFileSetAtRoot(root)
	require.NoError(t, err)

	inc, err := fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	excl, err := fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	s := &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err := s.GetFileList(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 9)

	inc, err = fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	excl, err = fileset.NewGlobSet(root, []string{"*.go"})
	require.NoError(t, err)

	s = &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err = s.GetFileList(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 1)

	inc, err = fileset.NewGlobSet(root, []string{"./.databricks/*.go"})
	require.NoError(t, err)

	excl, err = fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	s = &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err = s.GetFileList(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 10)
}

func TestRecursiveExclude(t *testing.T) {
	ctx := context.Background()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)
	fileSet, err := git.NewFileSetAtRoot(root)
	require.NoError(t, err)

	inc, err := fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	excl, err := fileset.NewGlobSet(root, []string{"test/**"})
	require.NoError(t, err)

	s := &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err := s.GetFileList(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 6)
}

func TestNegateExclude(t *testing.T) {
	ctx := context.Background()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)
	fileSet, err := git.NewFileSetAtRoot(root)
	require.NoError(t, err)

	inc, err := fileset.NewGlobSet(root, []string{})
	require.NoError(t, err)

	excl, err := fileset.NewGlobSet(root, []string{"./*", "!*.txt"})
	require.NoError(t, err)

	s := &Sync{
		SyncOptions: &SyncOptions{},

		fileSet:        fileSet,
		includeFileSet: inc,
		excludeFileSet: excl,
	}

	fileList, err := s.GetFileList(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 1)
	require.Equal(t, "test/sub1/sub2/h.txt", fileList[0].Relative)
}
