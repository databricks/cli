package vfs

import (
	"io/fs"
	"os"
	"path/filepath"
)

type osPath struct {
	path string

	openFn     func(name string) (fs.File, error)
	statFn     func(name string) (fs.FileInfo, error)
	readDirFn  func(name string) ([]fs.DirEntry, error)
	readFileFn func(name string) ([]byte, error)
}

func New(name string) Path {
	name = filepath.Clean(name)

	// [os.DirFS] implements all required interfaces.
	// We used type assertion below to get the underlying types.
	dirfs := os.DirFS(name)

	return &osPath{
		path: name,

		openFn:     dirfs.Open,
		statFn:     dirfs.(fs.StatFS).Stat,
		readDirFn:  dirfs.(fs.ReadDirFS).ReadDir,
		readFileFn: dirfs.(fs.ReadFileFS).ReadFile,
	}
}

func (o osPath) Open(name string) (fs.File, error) {
	return o.openFn(name)
}

func (o osPath) Stat(name string) (fs.FileInfo, error) {
	return o.statFn(name)
}

func (o osPath) ReadDir(name string) ([]fs.DirEntry, error) {
	return o.readDirFn(name)
}

func (o osPath) ReadFile(name string) ([]byte, error) {
	return o.readFileFn(name)
}

func (o osPath) Parent() Path {
	dir := filepath.Dir(o.path)
	if dir == o.path {
		return nil
	}

	return New(dir)
}

func (o osPath) Native() string {
	return o.path
}
