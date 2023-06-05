package filer

import (
	"bytes"
	"context"
	"io"
	"io/fs"
)

// filerFS implements the fs.FS interface for a filer.
type filerFS struct {
	ctx   context.Context
	filer Filer
}

// NewFS returns an fs.FS backed by a filer.
func NewFS(ctx context.Context, filer Filer) fs.FS {
	return &filerFS{ctx: ctx, filer: filer}
}

func (fs *filerFS) Open(name string) (fs.File, error) {
	stat, err := fs.filer.Stat(fs.ctx, name)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return &fsDir{fs: fs, name: name, stat: stat}, nil
	}

	return &fsFile{fs: fs, name: name, stat: stat}, nil
}

func (fs *filerFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.filer.ReadDir(fs.ctx, name)
}

func (fs *filerFS) ReadFile(name string) ([]byte, error) {
	reader, err := fs.filer.Read(fs.ctx, name)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (fs *filerFS) Stat(name string) (fs.FileInfo, error) {
	return fs.filer.Stat(fs.ctx, name)
}

// Type that implements fs.File for a filer-backed fs.FS.
type fsFile struct {
	fs   *filerFS
	name string
	stat fs.FileInfo

	reader io.Reader
}

func (f *fsFile) Stat() (fs.FileInfo, error) {
	return f.stat, nil
}

func (f *fsFile) Read(buf []byte) (int, error) {
	if f.reader == nil {
		reader, err := f.fs.filer.Read(f.fs.ctx, f.name)
		if err != nil {
			return 0, err
		}
		f.reader = reader
	}

	return f.reader.Read(buf)
}

func (f *fsFile) Close() error {
	if f.reader == nil {
		return fs.ErrClosed
	}
	f.reader = nil
	return nil
}

// Type that implements fs.ReadDirFile for a filer-backed fs.FS.
type fsDir struct {
	fs   *filerFS
	name string
	stat fs.FileInfo

	open    bool
	entries []fs.DirEntry
}

func (f *fsDir) Stat() (fs.FileInfo, error) {
	return f.stat, nil
}

func (f *fsDir) Read(buf []byte) (int, error) {
	return 0, fs.ErrInvalid
}

func (f *fsDir) ReadDir(n int) ([]fs.DirEntry, error) {
	// Load all directory entries if not already loaded.
	if !f.open {
		entries, err := f.fs.ReadDir(f.name)
		if err != nil {
			return nil, err
		}
		f.open = true
		f.entries = entries
	}

	// Return all entries if n <= 0.
	if n <= 0 {
		entries := f.entries
		f.entries = nil
		return entries, nil
	}

	// If there are no more entries, return io.EOF.
	if len(f.entries) == 0 {
		return nil, io.EOF
	}

	// If there are less than n entries, return all entries.
	if len(f.entries) < n {
		entries := f.entries
		f.entries = nil
		return entries, nil
	}

	// Return n entries.
	entries := f.entries[:n]
	f.entries = f.entries[n:]
	return entries, nil
}

func (f *fsDir) Close() error {
	if !f.open {
		return fs.ErrClosed
	}
	f.open = false
	f.entries = nil
	return nil
}
