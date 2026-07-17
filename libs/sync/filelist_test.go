package sync

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/fileset"
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

func relativePaths(files []fileset.File) []string {
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Relative)
	}
	slices.Sort(paths)
	return paths
}

func TestFileListFiles(t *testing.T) {
	ctx := t.Context()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)

	l, err := NewFileList(ctx, root, root, []string{"."}, []string{}, []string{})
	require.NoError(t, err)

	fileList, err := l.Files(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 9)

	l, err = NewFileList(ctx, root, root, []string{"."}, []string{}, []string{"*.go"})
	require.NoError(t, err)

	fileList, err = l.Files(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 1)

	l, err = NewFileList(ctx, root, root, []string{"."}, []string{"./.databricks/*.go"}, []string{})
	require.NoError(t, err)

	fileList, err = l.Files(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 10)
}

func TestRecursiveExclude(t *testing.T) {
	ctx := t.Context()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)

	l, err := NewFileList(ctx, root, root, []string{"."}, []string{}, []string{"test/**"})
	require.NoError(t, err)

	fileList, err := l.Files(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 6)
}

func TestNegateExclude(t *testing.T) {
	ctx := t.Context()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)

	l, err := NewFileList(ctx, root, root, []string{"."}, []string{}, []string{"./*", "!*.txt"})
	require.NoError(t, err)

	fileList, err := l.Files(ctx)
	require.NoError(t, err)
	require.Len(t, fileList, 1)
	require.Equal(t, "test/sub1/sub2/h.txt", fileList[0].Relative)
}

// TestFileListNestedRoot pins the two-root semantics: gitignore rules at the
// worktree root must govern a sync root nested below it. If NewFileList treated
// the sync root as the worktree root, the parent .gitignore would not be loaded
// and ignored.go would leak into the listing.
func TestFileListNestedRoot(t *testing.T) {
	ctx := t.Context()

	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, ".gitignore"), "ignored.go\n")
	testutil.Touch(t, dir, "sub", "keep.go")
	testutil.Touch(t, dir, "sub", "ignored.go")

	worktreeRoot := vfs.MustNew(dir)
	syncRoot := vfs.MustNew(filepath.Join(dir, "sub"))

	l, err := NewFileList(ctx, worktreeRoot, syncRoot, []string{"."}, nil, nil)
	require.NoError(t, err)

	fileList, err := l.Files(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"keep.go"}, relativePaths(fileList))
}

// TestFileListEmptyPathsIsNop pins that empty paths select no files: callers
// that want the whole root must say so with ["."]. There is no implicit default.
func TestFileListEmptyPathsIsNop(t *testing.T) {
	ctx := t.Context()

	dir := setupFiles(t)
	root := vfs.MustNew(dir)

	l, err := NewFileList(ctx, root, root, nil, nil, nil)
	require.NoError(t, err)
	files, err := l.Files(ctx)
	require.NoError(t, err)
	require.Empty(t, files)
}
