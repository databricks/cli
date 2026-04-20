package filer

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memFiler is a minimal in-memory implementation of libsfiler.Filer used to
// drive round-trip tests without a real workspace client. It models only the
// semantics the wrapper exercises (overwrite, create-parents, not-found).
type memFiler struct {
	files map[string][]byte // path -> content; directories tracked by prefix
	dirs  map[string]bool
}

func newMemFiler() *memFiler {
	return &memFiler{
		files: map[string][]byte{},
		dirs:  map[string]bool{"/": true, "": true},
	}
}

func (m *memFiler) Write(ctx context.Context, p string, r io.Reader, mode ...libsfiler.WriteMode) error {
	overwrite := slices.Contains(mode, libsfiler.OverwriteIfExists)
	createParents := slices.Contains(mode, libsfiler.CreateParentDirectories)

	if _, exists := m.files[p]; exists && !overwrite {
		return fs.ErrExist
	}
	parent := path.Dir(p)
	if !m.dirs[parent] {
		if !createParents {
			return fs.ErrNotExist
		}
		for d := parent; d != "/" && d != "." && d != ""; d = path.Dir(d) {
			m.dirs[d] = true
		}
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.files[p] = body
	return nil
}

func (m *memFiler) Read(ctx context.Context, p string) (io.ReadCloser, error) {
	body, ok := m.files[p]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (m *memFiler) Delete(ctx context.Context, p string, mode ...libsfiler.DeleteMode) error {
	if _, ok := m.files[p]; !ok {
		return fs.ErrNotExist
	}
	delete(m.files, p)
	return nil
}

func (m *memFiler) ReadDir(ctx context.Context, p string) ([]fs.DirEntry, error) {
	if !m.dirs[p] && p != "" {
		return nil, fs.ErrNotExist
	}
	var out []fs.DirEntry
	for filePath, body := range m.files {
		if path.Dir(filePath) != p {
			continue
		}
		out = append(out, memDirEntry{name: path.Base(filePath), size: int64(len(body))})
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (m *memFiler) Mkdir(ctx context.Context, p string) error {
	m.dirs[p] = true
	return nil
}

func (m *memFiler) Stat(ctx context.Context, p string) (fs.FileInfo, error) {
	if body, ok := m.files[p]; ok {
		return memFileInfo{name: path.Base(p), size: int64(len(body))}, nil
	}
	if m.dirs[p] {
		return memFileInfo{name: path.Base(p), isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

type memFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f memFileInfo) Name() string       { return f.name }
func (f memFileInfo) Size() int64        { return f.size }
func (f memFileInfo) Mode() fs.FileMode  { return 0 }
func (f memFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (f memFileInfo) IsDir() bool        { return f.isDir }
func (f memFileInfo) Sys() any           { return nil }

type memDirEntry struct {
	name string
	size int64
}

func (e memDirEntry) Name() string               { return e.name }
func (e memDirEntry) IsDir() bool                { return false }
func (e memDirEntry) Type() fs.FileMode          { return 0 }
func (e memDirEntry) Info() (fs.FileInfo, error) { return memFileInfo{name: e.name, size: e.size}, nil }

func newTestFiler(t *testing.T) (*WorkspaceFiler, *memFiler) {
	t.Helper()
	mem := newMemFiler()
	return newWorkspaceFilerFromInner(mem), mem
}

func TestWorkspaceFilerRoundTrip(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true

	require.NoError(t, f.Write(ctx, "/state/terraform.tfstate", strings.NewReader("hello"), 0))

	rc, err := f.Read(ctx, "/state/terraform.tfstate")
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(body))
}

func TestWorkspaceFilerWriteRejectsExistingWithoutOverwrite(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true

	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("v1"), 0))

	err := f.Write(ctx, "/state/a", strings.NewReader("v2"), 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrExist), "expected fs.ErrExist, got %v", err)
}

func TestWorkspaceFilerWriteOverwriteMode(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true

	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("v1"), 0))
	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("v2"), WriteModeOverwrite))

	rc, err := f.Read(ctx, "/state/a")
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "v2", string(body))
}

func TestWorkspaceFilerWriteCreateParents(t *testing.T) {
	ctx := t.Context()
	f, _ := newTestFiler(t)

	err := f.Write(ctx, "/state/nested/deep/a", strings.NewReader("x"), 0)
	require.Error(t, err, "write without CreateParents should fail for missing parent")
	assert.True(t, errors.Is(err, ErrNotFound))

	require.NoError(t, f.Write(ctx, "/state/nested/deep/a", strings.NewReader("x"), WriteModeCreateParents))

	rc, err := f.Read(ctx, "/state/nested/deep/a")
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "x", string(body))
}

func TestWorkspaceFilerReadMissingReturnsErrNotFound(t *testing.T) {
	ctx := t.Context()
	f, _ := newTestFiler(t)

	_, err := f.Read(ctx, "/state/missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}

func TestWorkspaceFilerStatMissingReturnsErrNotFound(t *testing.T) {
	ctx := t.Context()
	f, _ := newTestFiler(t)

	_, err := f.Stat(ctx, "/state/missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestWorkspaceFilerStatExistingFile(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true
	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("abcdef"), 0))

	info, err := f.Stat(ctx, "/state/a")
	require.NoError(t, err)
	assert.Equal(t, "a", info.Name())
	assert.Equal(t, int64(6), info.Size())
	assert.False(t, info.IsDir())
}

func TestWorkspaceFilerReadDir(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true
	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("1"), 0))
	require.NoError(t, f.Write(ctx, "/state/b", strings.NewReader("22"), 0))

	entries, err := f.ReadDir(ctx, "/state")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "a", entries[0].Name())
	assert.Equal(t, int64(1), entries[0].Size())
	assert.Equal(t, "b", entries[1].Name())
	assert.Equal(t, int64(2), entries[1].Size())
}

func TestWorkspaceFilerDeleteMissingIsNoop(t *testing.T) {
	ctx := t.Context()
	f, _ := newTestFiler(t)

	require.NoError(t, f.Delete(ctx, "/state/missing"))
}

func TestWorkspaceFilerDeleteExisting(t *testing.T) {
	ctx := t.Context()
	f, mem := newTestFiler(t)
	mem.dirs["/state"] = true
	require.NoError(t, f.Write(ctx, "/state/a", strings.NewReader("x"), 0))

	require.NoError(t, f.Delete(ctx, "/state/a"))
	_, err := f.Read(ctx, "/state/a")
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestWriteModeHas(t *testing.T) {
	cases := []struct {
		name  string
		m     WriteMode
		probe WriteMode
		want  bool
	}{
		{"zero has zero", 0, 0, true},
		{"overwrite has overwrite", WriteModeOverwrite, WriteModeOverwrite, true},
		{"overwrite lacks create-parents", WriteModeOverwrite, WriteModeCreateParents, false},
		{"combined has overwrite", WriteModeOverwrite | WriteModeCreateParents, WriteModeOverwrite, true},
		{"combined has create-parents", WriteModeOverwrite | WriteModeCreateParents, WriteModeCreateParents, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.m.Has(tc.probe))
		})
	}
}
