package testutil

import (
	"context"
	"io/fs"
	"testing"
	"time"

	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/mock"
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

func NewFakeDirEntry(name string, dir bool) fs.DirEntry {
	return fakeDirEntry{fakeFileInfo{name: name, dir: dir}}
}

func GetMockFilerForPath(t *testing.T, path string, entries []fs.DirEntry) func(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	mockFiler := mockfiler.NewMockFiler(t)
	if entries != nil {
		mockFiler.EXPECT().ReadDir(mock.AnythingOfType("*context.valueCtx"), path).Return(entries, nil)
	}
	mockFilerForPath := func(ctx context.Context, fullPath string) (filer.Filer, string, error) {
		return mockFiler, path, nil
	}
	return mockFilerForPath
}
