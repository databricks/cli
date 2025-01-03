package filer

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
)

var errNotImplemented = errors.New("not implemented")

type cacheTestFiler struct {
	calls int

	readDir map[string][]fs.DirEntry
	stat    map[string]fs.FileInfo
}

func (m *cacheTestFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...WriteMode) error {
	return errNotImplemented
}

func (m *cacheTestFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (m *cacheTestFiler) Delete(ctx context.Context, path string, mode ...DeleteMode) error {
	return errNotImplemented
}

func (m *cacheTestFiler) ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error) {
	m.calls++
	if fi, ok := m.readDir[path]; ok {
		delete(m.readDir, path)
		return fi, nil
	}
	return nil, fs.ErrNotExist
}

func (m *cacheTestFiler) Mkdir(ctx context.Context, path string) error {
	return errNotImplemented
}

func (m *cacheTestFiler) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	m.calls++
	if fi, ok := m.stat[name]; ok {
		delete(m.stat, name)
		return fi, nil
	}
	return nil, fs.ErrNotExist
}

func TestWorkspaceFilesCache_ReadDirCache(t *testing.T) {
	f := &cacheTestFiler{
		readDir: map[string][]fs.DirEntry{
			"dir1": {
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file1",
							Size:       1,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file2",
							Size:       2,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	cache := newWorkspaceFilesReadaheadCache(f)
	defer cache.Cleanup()

	// First read dir should hit the filer, second should hit the cache.
	for range 2 {
		fi, err := cache.ReadDir(ctx, "dir1")
		assert.NoError(t, err)
		if assert.Len(t, fi, 2) {
			assert.Equal(t, "file1", fi[0].Name())
			assert.Equal(t, "file2", fi[1].Name())
		}
	}

	// Third stat should hit the filer, fourth should hit the cache.
	for range 2 {
		_, err := cache.ReadDir(ctx, "dir2")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	}

	// Assert we only called the filer twice.
	assert.Equal(t, 2, f.calls)
}

func TestWorkspaceFilesCache_ReadDirCacheIsolation(t *testing.T) {
	f := &cacheTestFiler{
		readDir: map[string][]fs.DirEntry{
			"dir": {
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file",
							Size:       1,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	cache := newWorkspaceFilesReadaheadCache(f)
	defer cache.Cleanup()

	// First read dir should hit the filer, second should hit the cache.
	entries, err := cache.ReadDir(ctx, "dir")
	assert.NoError(t, err)
	assert.Equal(t, "file", entries[0].Name())

	// Modify the entry to check that mutations are not reflected in the cache.
	entries[0] = wsfsDirEntry{
		wsfsFileInfo{
			ObjectInfo: workspace.ObjectInfo{
				Path: "tainted",
			},
		},
	}

	// Read the directory again to check that the cache is isolated.
	entries, err = cache.ReadDir(ctx, "dir")
	assert.NoError(t, err)
	assert.Equal(t, "file", entries[0].Name())
}

func TestWorkspaceFilesCache_StatCache(t *testing.T) {
	f := &cacheTestFiler{
		stat: map[string]fs.FileInfo{
			"file1": &wsfsFileInfo{ObjectInfo: workspace.ObjectInfo{Path: "file1", Size: 1}},
		},
	}

	ctx := context.Background()
	cache := newWorkspaceFilesReadaheadCache(f)
	defer cache.Cleanup()

	// First stat should hit the filer, second should hit the cache.
	for range 2 {
		fi, err := cache.Stat(ctx, "file1")
		if assert.NoError(t, err) {
			assert.Equal(t, "file1", fi.Name())
			assert.Equal(t, int64(1), fi.Size())
		}
	}

	// Third stat should hit the filer, fourth should hit the cache.
	for range 2 {
		_, err := cache.Stat(ctx, "file2")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	}

	// Assert we only called the filer twice.
	assert.Equal(t, 2, f.calls)
}

func TestWorkspaceFilesCache_ReadDirPopulatesStatCache(t *testing.T) {
	f := &cacheTestFiler{
		readDir: map[string][]fs.DirEntry{
			"dir1": {
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file1",
							Size:       1,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file2",
							Size:       2,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "notebook1",
							Size:       1,
							ObjectType: workspace.ObjectTypeNotebook,
						},
						ReposExportFormat: "this should not end up in the stat cache",
					},
				},
			},
		},
		stat: map[string]fs.FileInfo{
			"dir1/notebook1": wsfsFileInfo{
				ObjectInfo: workspace.ObjectInfo{
					Path:       "notebook1",
					Size:       1,
					ObjectType: workspace.ObjectTypeNotebook,
				},
				ReposExportFormat: workspace.ExportFormatJupyter,
			},
		},
	}

	ctx := context.Background()
	cache := newWorkspaceFilesReadaheadCache(f)
	defer cache.Cleanup()

	// Issue read dir to populate the stat cache.
	_, err := cache.ReadDir(ctx, "dir1")
	assert.NoError(t, err)

	// Stat on a file in the directory should hit the cache.
	fi, err := cache.Stat(ctx, "dir1/file1")
	if assert.NoError(t, err) {
		assert.Equal(t, "file1", fi.Name())
		assert.Equal(t, int64(1), fi.Size())
	}

	// If the containing directory has been read, absence is also inferred from the cache.
	_, err = cache.Stat(ctx, "dir1/file3")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Stat on a notebook in the directory should have been performed in the background.
	fi, err = cache.Stat(ctx, "dir1/notebook1")
	if assert.NoError(t, err) {
		assert.Equal(t, "notebook1", fi.Name())
		assert.Equal(t, int64(1), fi.Size())
		assert.Equal(t, workspace.ExportFormatJupyter, fi.(wsfsFileInfo).ReposExportFormat)
	}

	// Assert we called the filer twice (once for read dir, once for stat on the notebook).
	assert.Equal(t, 2, f.calls)
}

func TestWorkspaceFilesCache_ReadDirTriggersReadahead(t *testing.T) {
	f := &cacheTestFiler{
		readDir: map[string][]fs.DirEntry{
			"a": {
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "b1",
							ObjectType: workspace.ObjectTypeDirectory,
						},
					},
				},
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "b2",
							ObjectType: workspace.ObjectTypeDirectory,
						},
					},
				},
			},
			"a/b1": {
				wsfsDirEntry{
					wsfsFileInfo{
						ObjectInfo: workspace.ObjectInfo{
							Path:       "file1",
							Size:       1,
							ObjectType: workspace.ObjectTypeFile,
						},
					},
				},
			},
			"a/b2": {},
		},
	}

	ctx := context.Background()
	cache := newWorkspaceFilesReadaheadCache(f)
	defer cache.Cleanup()

	// Issue read dir to populate the stat cache.
	_, err := cache.ReadDir(ctx, "a")
	assert.NoError(t, err)

	// Stat on a directory in the directory should hit the cache.
	fi, err := cache.Stat(ctx, "a/b1")
	if assert.NoError(t, err) {
		assert.Equal(t, "b1", fi.Name())
		assert.True(t, fi.IsDir())
	}

	// Stat on a file in a nested directory should hit the cache.
	fi, err = cache.Stat(ctx, "a/b1/file1")
	if assert.NoError(t, err) {
		assert.Equal(t, "file1", fi.Name())
		assert.Equal(t, int64(1), fi.Size())
	}

	// Stat on a non-existing file in an empty nested directory should hit the cache.
	_, err = cache.Stat(ctx, "a/b2/file2")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// Assert we called the filer 3 times; once for each directory.
	assert.Equal(t, 3, f.calls)
}
