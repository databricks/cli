package fakefs

import (
	"io/fs"
	"time"
)

// DirEntry is a fake implementation of [fs.DirEntry].
type DirEntry struct {
	FileInfo
}

func (entry DirEntry) Type() fs.FileMode {
	typ := fs.ModePerm
	if entry.FakeDir {
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
