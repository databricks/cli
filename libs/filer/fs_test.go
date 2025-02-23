package filer

import (
	"context"
	"io"
	"io/fs"
	"testing"

	"github.com/databricks/cli/libs/fakefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	fakeFiler := NewFakeFiler(map[string]fakefs.FileInfo{
		".":     {FakeName: "root", FakeDir: true},
		"dirA":  {FakeDir: true},
		"dirB":  {FakeDir: true},
		"fileA": {FakeSize: 3},
	})

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
	assert.False(t, info.IsDir())

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
	assert.True(t, info.IsDir())

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

		for range 3 {
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
