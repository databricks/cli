package fs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
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

func TestCp_cpDirToDir_contextCancellation(t *testing.T) {
	testError := errors.New("test error")

	// Mock the stats and readDir methods for a Filer over a file system that
	// has the following directory structure:
	//
	//   src/
	//   ├── subdir/
	//   ├── file1.txt
	//   ├── file2.txt
	//   └── file3.txt
	//
	mockSourceStat := func(ctx context.Context, path string) (fs.FileInfo, error) {
		isDir := path == "src" || path == "src/subdir"
		return mockFileInfo{name: path, isDir: isDir}, nil
	}
	mockSourceReadDir := func(ctx context.Context, path string) ([]fs.DirEntry, error) {
		if path == "src" {
			return []fs.DirEntry{
				mockDirEntry{name: "subdir", isDir: true},
				mockDirEntry{name: "file1.txt", isDir: false},
				mockDirEntry{name: "file2.txt", isDir: false},
				mockDirEntry{name: "file3.txt", isDir: false},
			}, nil
		}
		return nil, nil
	}

	testCases := []struct {
		desc    string
		c       *copy
		wantErr error
	}{
		{
			// The source filer's Read method blocks until context is cancelled,
			// simulating a slow file copy operation. The target filer's Mkdir
			// method returns an error which should cancel the walk and all file
			// copy goroutines.
			desc: "cancel go routines on walk error",
			c: &copy{
				recursive:   true,
				concurrency: 5,
				sourceFiler: &mockFiler{
					stat:    mockSourceStat,
					readDir: mockSourceReadDir,
					read: func(ctx context.Context, path string) (io.ReadCloser, error) {
						<-ctx.Done() // block until context is cancelled
						return nil, ctx.Err()
					},
				},
				targetFiler: &mockFiler{
					mkdir: func(ctx context.Context, path string) error {
						return testError
					},
				},
			},
			wantErr: testError,
		},
		{
			// The target filer's Write method returns an error when writing the
			// file1.txt file. This error is expected to be returned by the file copy
			// goroutine and all other file copy goroutines should be cancelled.
			desc: "cancel go routines on file copy error",
			c: &copy{
				recursive:   true,
				concurrency: 5,
				sourceFiler: &mockFiler{
					stat:    mockSourceStat,
					readDir: mockSourceReadDir,
					read: func(ctx context.Context, path string) (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader("content")), nil
					},
				},
				targetFiler: &mockFiler{
					write: func(ctx context.Context, path string, r io.Reader, mode ...filer.WriteMode) error {
						if path == "dst/file1.txt" {
							return testError
						}
						<-ctx.Done() // block until context is cancelled
						return ctx.Err()
					},
				},
			},
			wantErr: testError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			done := make(chan error, 1)
			go func() {
				done <- tc.c.cpDirToDir(t.Context(), "src", "dst")
			}()

			select {
			case gotErr := <-done:
				if !errors.Is(gotErr, tc.wantErr) {
					t.Errorf("want error %v, got %v", tc.wantErr, gotErr)
				}
			case <-time.After(3 * time.Second): // do not wait too long in case of test issues
				t.Fatal("cpDirToDir blocked instead of returning error immediately")
			}
		})
	}
}
