package filer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

type FakeDirEntry struct {
	FakeFileInfo
}

func (entry FakeDirEntry) Type() fs.FileMode {
	typ := fs.ModePerm
	if entry.FakeDir {
		typ |= fs.ModeDir
	}
	return typ
}

func (entry FakeDirEntry) Info() (fs.FileInfo, error) {
	return entry.FakeFileInfo, nil
}

type FakeFileInfo struct {
	FakeName string
	FakeSize int64
	FakeDir  bool
	FakeMode fs.FileMode
}

func (info FakeFileInfo) Name() string {
	return info.FakeName
}

func (info FakeFileInfo) Size() int64 {
	return info.FakeSize
}

func (info FakeFileInfo) Mode() fs.FileMode {
	return info.FakeMode
}

func (info FakeFileInfo) ModTime() time.Time {
	return time.Now()
}

func (info FakeFileInfo) IsDir() bool {
	return info.FakeDir
}

func (info FakeFileInfo) Sys() any {
	return nil
}

type FakeFiler struct {
	entries map[string]FakeFileInfo
}

func (f *FakeFiler) Write(ctx context.Context, p string, reader io.Reader, mode ...WriteMode) error {
	return fmt.Errorf("not implemented")
}

func (f *FakeFiler) Read(ctx context.Context, p string) (io.ReadCloser, error) {
	_, ok := f.entries[p]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return io.NopCloser(strings.NewReader("foo")), nil
}

func (f *FakeFiler) Delete(ctx context.Context, p string, mode ...DeleteMode) error {
	return fmt.Errorf("not implemented")
}

func (f *FakeFiler) ReadDir(ctx context.Context, p string) ([]fs.DirEntry, error) {
	p = strings.TrimSuffix(p, "/")
	entry, ok := f.entries[p]
	if !ok {
		return nil, NoSuchDirectoryError{p}
	}

	if !entry.FakeDir {
		return nil, fs.ErrInvalid
	}

	// Find all entries contained in the specified directory `p`.
	var out []fs.DirEntry
	for k, v := range f.entries {
		if k == p || path.Dir(k) != p {
			continue
		}

		out = append(out, FakeDirEntry{v})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (f *FakeFiler) Mkdir(ctx context.Context, path string) error {
	return fmt.Errorf("not implemented")
}

func (f *FakeFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	entry, ok := f.entries[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return entry, nil
}

func NewFakeFiler(entries map[string]FakeFileInfo) *FakeFiler {
	fakeFiler := &FakeFiler{
		entries: entries,
	}

	for k, v := range fakeFiler.entries {
		if v.FakeName != "" {
			continue
		}
		v.FakeName = path.Base(k)
		fakeFiler.entries[k] = v
	}

	return fakeFiler
}
