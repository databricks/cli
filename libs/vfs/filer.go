package vfs

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
)

type filerPath struct {
	ctx  context.Context
	path string
	fs   FS

	construct func(path string) (filer.Filer, error)
}

func NewFilerPath(ctx context.Context, path string, construct func(path string) (filer.Filer, error)) (Path, error) {
	f, err := construct(path)
	if err != nil {
		return nil, err
	}

	return &filerPath{
		ctx:  ctx,
		path: path,
		fs:   filer.NewFS(ctx, f).(FS),

		construct: construct,
	}, nil
}

func (f filerPath) Open(name string) (fs.File, error) {
	return f.fs.Open(name)
}

func (f filerPath) Stat(name string) (fs.FileInfo, error) {
	return f.fs.Stat(name)
}

func (f filerPath) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.fs.ReadDir(name)
}

func (f filerPath) ReadFile(name string) ([]byte, error) {
	return f.fs.ReadFile(name)
}

func (f filerPath) Parent() Path {
	if f.path == "/" {
		return nil
	}

	dir := filepath.Dir(f.path)
	nf, err := NewFilerPath(f.ctx, dir, f.construct)
	if err != nil {
		panic(err)
	}

	return nf
}

func (f filerPath) Native() string {
	return f.path
}
