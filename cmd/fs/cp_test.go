package fs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFiler mocks filer.Filer.
type mockFiler struct {
	write   func(ctx context.Context, path string, r io.Reader, mode ...filer.WriteMode) error
	read    func(ctx context.Context, path string) (io.ReadCloser, error)
	delete  func(ctx context.Context, path string, mode ...filer.DeleteMode) error
	readDir func(ctx context.Context, path string) ([]fs.DirEntry, error)
	mkdir   func(ctx context.Context, path string) error
	stat    func(ctx context.Context, path string) (fs.FileInfo, error)
}

func (m *mockFiler) Write(ctx context.Context, path string, r io.Reader, mode ...filer.WriteMode) error {
	if m.write == nil {
		return nil
	}
	return m.write(ctx, path, r, mode...)
}

func (m *mockFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.read == nil {
		return nil, nil
	}
	return m.read(ctx, path)
}

func (m *mockFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	if m.delete == nil {
		return nil
	}
	return m.delete(ctx, path, mode...)
}

func (m *mockFiler) ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error) {
	if m.readDir == nil {
		return nil, nil
	}
	return m.readDir(ctx, path)
}

func (m *mockFiler) Mkdir(ctx context.Context, path string) error {
	if m.mkdir == nil {
		return nil
	}
	return m.mkdir(ctx, path)
}

func (m *mockFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	if m.stat == nil {
		return nil, nil
	}
	return m.stat(ctx, path)
}

// mockFileInfo mocks fs.FileInfo.
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() fs.FileMode  { return 0o644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() any           { return nil }

// mockDirEntry mocks fs.DirEntry.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string      { return m.name }
func (m mockDirEntry) IsDir() bool       { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) {
	return mockFileInfo(m), nil
}

func TestCp_writeModes(t *testing.T) {
	// src/
	// ├── file1.txt
	// └── subdir/
	//     └── file2.txt
	sourceFiler := &mockFiler{
		stat: func(ctx context.Context, path string) (fs.FileInfo, error) {
			isDir := path == "src" || path == "src/subdir"
			return mockFileInfo{name: path, isDir: isDir}, nil
		},
		readDir: func(ctx context.Context, path string) ([]fs.DirEntry, error) {
			switch path {
			case "src":
				return []fs.DirEntry{
					mockDirEntry{name: "file1.txt"},
					mockDirEntry{name: "subdir", isDir: true},
				}, nil
			case "src/subdir":
				return []fs.DirEntry{mockDirEntry{name: "file2.txt"}}, nil
			}
			return nil, nil
		},
		read: func(ctx context.Context, path string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("content")), nil
		},
	}

	// The target filer records the modes of each write, and fails a write whose
	// directory the walk has not created yet: skipping the check is only safe
	// because of that Mkdir.
	newTargetFiler := func(got map[string][]filer.WriteMode) *mockFiler {
		var mu sync.Mutex
		created := map[string]bool{}
		return &mockFiler{
			mkdir: func(ctx context.Context, dir string) error {
				mu.Lock()
				defer mu.Unlock()
				created[dir] = true
				return nil
			},
			write: func(ctx context.Context, p string, r io.Reader, mode ...filer.WriteMode) error {
				mu.Lock()
				defer mu.Unlock()
				if slices.Contains(mode, filer.CreateParentDirectories) && !created[path.Dir(p)] {
					return fmt.Errorf("write to %s before Mkdir of its directory", p)
				}
				got[p] = mode
				return nil
			},
		}
	}

	for _, overwrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("overwrite=%t", overwrite), func(t *testing.T) {
			want := []filer.WriteMode{filer.CreateParentDirectories}
			if overwrite {
				want = append(want, filer.OverwriteIfExists)
			}

			// A recursive copy creates each target directory during the walk,
			// so it skips the per-file parent-directory check.
			gotDir := map[string][]filer.WriteMode{}
			c := &copy{
				recursive:   true,
				overwrite:   overwrite,
				concurrency: 2,
				sourceFiler: sourceFiler,
				targetFiler: newTargetFiler(gotDir),
			}
			require.NoError(t, c.cpDirToDir(cmdio.MockDiscard(t.Context()), "src", "dst"))
			assert.Equal(t, map[string][]filer.WriteMode{
				"dst/file1.txt":        want,
				"dst/subdir/file2.txt": want,
			}, gotDir)

			// A single-file copy keeps the check: its target directory was not
			// created by this command.
			gotFile := map[string][]filer.WriteMode{}
			c.targetFiler = newTargetFiler(gotFile)
			require.NoError(t, c.cpFileToFile(cmdio.MockDiscard(t.Context()), "src/file1.txt", "dst/file1.txt"))
			var wantFile []filer.WriteMode
			if overwrite {
				wantFile = []filer.WriteMode{filer.OverwriteIfExists}
			}
			assert.Equal(t, map[string][]filer.WriteMode{"dst/file1.txt": wantFile}, gotFile)
		})
	}
}
