package fileset

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
)

func TestFileSet_NoPaths(t *testing.T) {
	fs := New(vfs.MustNew("testdata"))
	files, err := fs.Files()
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, files, 4)
	assert.Equal(t, "dir1/a", files[0].Relative)
	assert.Equal(t, "dir1/b", files[1].Relative)
	assert.Equal(t, "dir2/a", files[2].Relative)
	assert.Equal(t, "dir2/b", files[3].Relative)
}

func TestFileSet_ParentPath(t *testing.T) {
	fs := New(vfs.MustNew("testdata"), []string{".."})
	_, err := fs.Files()

	// It is impossible to escape the root directory.
	assert.Error(t, err)
}

func TestFileSet_DuplicatePaths(t *testing.T) {
	fs := New(vfs.MustNew("testdata"), []string{"dir1", "dir1"})
	files, err := fs.Files()
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, files, 2)
	assert.Equal(t, "dir1/a", files[0].Relative)
	assert.Equal(t, "dir1/b", files[1].Relative)
}

func TestFileSet_OverlappingPaths(t *testing.T) {
	fs := New(vfs.MustNew("testdata"), []string{"dir1", "dir1/a"})
	files, err := fs.Files()
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, files, 2)
	assert.Equal(t, "dir1/a", files[0].Relative)
	assert.Equal(t, "dir1/b", files[1].Relative)
}

func TestFileSet_IgnoreDirError(t *testing.T) {
	testError := errors.New("test error")
	fs := New(vfs.MustNew("testdata"))
	fs.SetIgnorer(testIgnorer{dirErr: testError})
	_, err := fs.Files()
	assert.ErrorIs(t, err, testError)
}

func TestFileSet_IgnoreDir(t *testing.T) {
	fs := New(vfs.MustNew("testdata"))
	fs.SetIgnorer(testIgnorer{dir: []string{"dir1"}})
	files, err := fs.Files()
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, files, 2)
	assert.Equal(t, "dir2/a", files[0].Relative)
	assert.Equal(t, "dir2/b", files[1].Relative)
}

func TestFileSet_IgnoreFileError(t *testing.T) {
	testError := errors.New("test error")
	fs := New(vfs.MustNew("testdata"))
	fs.SetIgnorer(testIgnorer{fileErr: testError})
	_, err := fs.Files()
	assert.ErrorIs(t, err, testError)
}

func TestFileSet_IgnoreFile(t *testing.T) {
	fs := New(vfs.MustNew("testdata"))
	fs.SetIgnorer(testIgnorer{file: []string{"dir1/a"}})
	files, err := fs.Files()
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, files, 3)
	assert.Equal(t, "dir1/b", files[0].Relative)
	assert.Equal(t, "dir2/a", files[1].Relative)
	assert.Equal(t, "dir2/b", files[2].Relative)
}

type testIgnorer struct {
	// dir is a list of directories to ignore. Strings are compared verbatim.
	dir []string

	// dirErr is an error to return when IgnoreDirectory is called.
	dirErr error

	// file is a list of files to ignore. Strings are compared verbatim.
	file []string

	// fileErr is an error to return when IgnoreFile is called.
	fileErr error
}

// IgnoreDirectory returns true if the path is in the dir list.
// If dirErr is set, it returns dirErr.
func (t testIgnorer) IgnoreDirectory(path string) (bool, error) {
	if t.dirErr != nil {
		return false, t.dirErr
	}

	for _, d := range t.dir {
		if d == path {
			return true, nil
		}
	}

	return false, nil
}

// IgnoreFile returns true if the path is in the file list.
// If fileErr is set, it returns fileErr.
func (t testIgnorer) IgnoreFile(path string) (bool, error) {
	if t.fileErr != nil {
		return false, t.fileErr
	}

	for _, f := range t.file {
		if f == path {
			return true, nil
		}
	}

	return false, nil
}
