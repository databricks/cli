package filer

import (
	"context"
	"io"
	"io/fs"
	"testing"

	"github.com/databricks/cli/libs/fakefs"
	"github.com/stretchr/testify/assert"
)

func TestFakeFiler_Read(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"file": {},
	})

	ctx := context.Background()
	r, err := f.Read(ctx, "file")
	if !assert.NoError(t, err) {
		return
	}

	contents, err := io.ReadAll(r)
	if !assert.NoError(t, err) {
		return
	}

	// Contents of every file is "foo".
	assert.Equal(t, "foo", string(contents))
}

func TestFakeFiler_Read_NotFound(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"foo": {},
	})

	ctx := context.Background()
	_, err := f.Read(ctx, "bar")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFakeFiler_ReadDir_NotFound(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"dir1": {FakeDir: true},
	})

	ctx := context.Background()
	_, err := f.ReadDir(ctx, "dir2")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFakeFiler_ReadDir_NotADirectory(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"file": {},
	})

	ctx := context.Background()
	_, err := f.ReadDir(ctx, "file")
	assert.ErrorIs(t, err, fs.ErrInvalid)
}

func TestFakeFiler_ReadDir(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"dir1":       {FakeDir: true},
		"dir1/file2": {},
		"dir1/dir2":  {FakeDir: true},
	})

	ctx := context.Background()
	entries, err := f.ReadDir(ctx, "dir1/")
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Len(t, entries, 2) {
		return
	}

	// The entries are sorted by name.
	assert.Equal(t, "dir2", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "file2", entries[1].Name())
	assert.False(t, entries[1].IsDir())
}

func TestFakeFiler_Stat(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"file": {},
	})

	ctx := context.Background()
	info, err := f.Stat(ctx, "file")
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "file", info.Name())
}

func TestFakeFiler_Stat_NotFound(t *testing.T) {
	f := NewFakeFiler(map[string]fakefs.FileInfo{
		"foo": {},
	})

	ctx := context.Background()
	_, err := f.Stat(ctx, "bar")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
