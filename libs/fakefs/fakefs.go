package fakefs

import (
	"errors"
	"io/fs"
	"time"
)

var ErrNotImplemented = errors.New("not implemented")

// DirEntry is a fake implementation of [fs.DirEntry].
type DirEntry struct {
	fs.FileInfo
}

func (entry DirEntry) Type() fs.FileMode {
	typ := fs.ModePerm
	if entry.IsDir() {
		typ |= fs.ModeDir
	}
	return typ
}

func (entry DirEntry) Info() (fs.FileInfo, error) {
	return entry.FileInfo, nil
}

// FileInfo is a fake implementation of [fs.FileInfo].
type FileInfo struct {
	FakeName string
	FakeSize int64
	FakeDir  bool
	FakeMode fs.FileMode
}

func (info FileInfo) Name() string {
	return info.FakeName
}

func (info FileInfo) Size() int64 {
	return info.FakeSize
}

func (info FileInfo) Mode() fs.FileMode {
	return info.FakeMode
}

func (info FileInfo) ModTime() time.Time {
	return time.Now()
}

func (info FileInfo) IsDir() bool {
	return info.FakeDir
}

func (info FileInfo) Sys() any {
	return nil
}

// File is a fake implementation of [fs.File].
type File struct {
	FileInfo fs.FileInfo
}

func (f File) Close() error {
	return nil
}

func (f File) Read(p []byte) (n int, err error) {
	return 0, ErrNotImplemented
}

func (f File) Stat() (fs.FileInfo, error) {
	return f.FileInfo, nil
}

// FS is a fake implementation of [fs.FS].
type FS map[string]fs.File

func (f FS) Open(name string) (fs.File, error) {
	e, ok := f[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return e, nil
}
