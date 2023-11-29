package filer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDirEntry struct {
	fakeFileInfo
}

func (entry fakeDirEntry) Type() fs.FileMode {
	typ := fs.ModePerm
	if entry.dir {
		typ |= fs.ModeDir
	}
	return typ
}

func (entry fakeDirEntry) Info() (fs.FileInfo, error) {
	return entry.fakeFileInfo, nil
}

type fakeFileInfo struct {
	name string
	size int64
	dir  bool
	mode fs.FileMode
}

func (info fakeFileInfo) Name() string {
	return info.name
}

func (info fakeFileInfo) Size() int64 {
	return info.size
}

func (info fakeFileInfo) Mode() fs.FileMode {
	return info.mode
}

func (info fakeFileInfo) ModTime() time.Time {
	return time.Now()
}

func (info fakeFileInfo) IsDir() bool {
	return info.dir
}

func (info fakeFileInfo) Sys() any {
	return nil
}

type fakeFiler struct {
	entries map[string]fakeFileInfo
}

func (f *fakeFiler) Write(ctx context.Context, p string, reader io.Reader, size int64, mode ...WriteMode) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeFiler) Read(ctx context.Context, p string) (io.ReadCloser, error) {
	_, ok := f.entries[p]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return io.NopCloser(strings.NewReader("foo")), nil
}

func (f *fakeFiler) Delete(ctx context.Context, p string, mode ...DeleteMode) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeFiler) ReadDir(ctx context.Context, p string) ([]fs.DirEntry, error) {
	entry, ok := f.entries[p]
	if !ok {
		return nil, fs.ErrNotExist
	}

	if !entry.dir {
		return nil, fs.ErrInvalid
	}

	// Find all entries contained in the specified directory `p`.
	var out []fs.DirEntry
	for k, v := range f.entries {
		if k == p || path.Dir(k) != p {
			continue
		}

		out = append(out, fakeDirEntry{v})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (f *fakeFiler) Mkdir(ctx context.Context, path string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	entry, ok := f.entries[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return entry, nil
}

func TestFsImplementsFS(t *testing.T) {
	var _ fs.FS = &filerFS{}
}

func TestFsImplementsReadDirFS(t *testing.T) {
	var _ fs.ReadDirFS = &filerFS{}
}

func TestFsImplementsReadFileFS(t *testing.T) {
	var _ fs.ReadDirFS = &filerFS{}
}

func TestFsImplementsStatFS(t *testing.T) {
	var _ fs.StatFS = &filerFS{}
}

func TestFsFileImplementsFsFile(t *testing.T) {
	var _ fs.File = &fsFile{}
}

func TestFsDirImplementsFsReadDirFile(t *testing.T) {
	var _ fs.ReadDirFile = &fsDir{}
}

func fakeFS() fs.FS {
	fakeFiler := &fakeFiler{
		entries: map[string]fakeFileInfo{
			".":     {name: "root", dir: true},
			"dirA":  {dir: true},
			"dirB":  {dir: true},
			"fileA": {size: 3},
		},
	}

	for k, v := range fakeFiler.entries {
		if v.name != "" {
			continue
		}
		v.name = path.Base(k)
		fakeFiler.entries[k] = v
	}

	return NewFS(context.Background(), fakeFiler)
}

func TestFsGlob(t *testing.T) {
	fakeFS := fakeFS()
	matches, err := fs.Glob(fakeFS, "*")
	require.NoError(t, err)
	assert.Equal(t, []string{"dirA", "dirB", "fileA"}, matches)
}

func TestFsOpenFile(t *testing.T) {
	fakeFS := fakeFS()
	fakeFile, err := fakeFS.Open("fileA")
	require.NoError(t, err)

	info, err := fakeFile.Stat()
	require.NoError(t, err)
	assert.Equal(t, "fileA", info.Name())
	assert.Equal(t, int64(3), info.Size())
	assert.Equal(t, fs.FileMode(0), info.Mode())
	assert.Equal(t, false, info.IsDir())

	// Read until closed.
	b := make([]byte, 3)
	n, err := fakeFile.Read(b)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte{'f', 'o', 'o'}, b)
	_, err = fakeFile.Read(b)
	assert.ErrorIs(t, err, io.EOF)

	// Close.
	err = fakeFile.Close()
	assert.NoError(t, err)

	// Close again.
	err = fakeFile.Close()
	assert.ErrorIs(t, err, fs.ErrClosed)
}

func TestFsOpenDir(t *testing.T) {
	fakeFS := fakeFS()
	fakeFile, err := fakeFS.Open(".")
	require.NoError(t, err)

	info, err := fakeFile.Stat()
	require.NoError(t, err)
	assert.Equal(t, "root", info.Name())
	assert.Equal(t, true, info.IsDir())

	de, ok := fakeFile.(fs.ReadDirFile)
	require.True(t, ok)

	// Read all entries in one shot.
	reference, err := de.ReadDir(-1)
	require.NoError(t, err)

	// Read entries one at a time.
	{
		var tmp, entries []fs.DirEntry
		var err error

		de.Close()

		for i := 0; i < 3; i++ {
			tmp, err = de.ReadDir(1)
			require.NoError(t, err)
			entries = append(entries, tmp...)
		}

		_, err = de.ReadDir(1)
		require.ErrorIs(t, err, io.EOF, err)

		// Compare to reference.
		assert.Equal(t, reference, entries)
	}

	// Read entries and overshoot at the end.
	{
		var tmp, entries []fs.DirEntry
		var err error

		de.Close()

		tmp, err = de.ReadDir(1)
		require.NoError(t, err)
		entries = append(entries, tmp...)

		tmp, err = de.ReadDir(20)
		require.NoError(t, err)
		entries = append(entries, tmp...)

		_, err = de.ReadDir(1)
		require.ErrorIs(t, err, io.EOF, err)

		// Compare to reference.
		assert.Equal(t, reference, entries)
	}
}

func TestFsReadDir(t *testing.T) {
	fakeFS := fakeFS().(fs.ReadDirFS)
	entries, err := fakeFS.ReadDir(".")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "dirA", entries[0].Name())
	assert.Equal(t, "dirB", entries[1].Name())
	assert.Equal(t, "fileA", entries[2].Name())
}

func TestFsReadFile(t *testing.T) {
	fakeFS := fakeFS().(fs.ReadFileFS)
	buf, err := fakeFS.ReadFile("fileA")
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), buf)
}

func TestFsStat(t *testing.T) {
	fakeFS := fakeFS().(fs.StatFS)
	info, err := fakeFS.Stat("fileA")
	require.NoError(t, err)
	assert.Equal(t, "fileA", info.Name())
	assert.Equal(t, int64(3), info.Size())
}
