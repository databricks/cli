package filer

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/fakefs"
)

type FakeFiler struct {
	entries map[string]fakefs.FileInfo
}

func (f *FakeFiler) Write(ctx context.Context, p string, reader io.Reader, mode ...WriteMode) error {
	return errors.New("not implemented")
}

func (f *FakeFiler) Read(ctx context.Context, p string) (io.ReadCloser, error) {
	_, ok := f.entries[p]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return io.NopCloser(strings.NewReader("foo")), nil
}

func (f *FakeFiler) Delete(ctx context.Context, p string, mode ...DeleteMode) error {
	return errors.New("not implemented")
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

		out = append(out, fakefs.DirEntry{FileInfo: v})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (f *FakeFiler) Mkdir(ctx context.Context, path string) error {
	return errors.New("not implemented")
}

func (f *FakeFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	entry, ok := f.entries[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return entry, nil
}

// NewFakeFiler creates a new fake [Filer] instance with the given entries.
// It sets the [Name] field of each entry to the base name of the path.
//
// This is meant to be used in tests.
func NewFakeFiler(entries map[string]fakefs.FileInfo) *FakeFiler {
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
