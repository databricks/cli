// Package filer defines the pluggable state-storage backend used by UCM.
//
// StateFiler abstracts the small set of file operations needed by the state
// manager (U4) and the terraform wrapper (U5): read, write, delete, stat,
// and listing. The v1 implementation is backed by workspace files
// (see workspace_filer.go); v2 adds S3/ADLS/GCS backends.
//
// Paths are always forward-slash-separated and relative to the root configured
// on the concrete implementation.
package filer

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotFound is returned by Read and Stat when the target path does not exist.
// Concrete implementations may return wrapped errors; callers must use
// errors.Is to compare.
var ErrNotFound = errors.New("ucm filer: path not found")

// WriteMode is a bit-mask controlling the behavior of StateFiler.Write.
type WriteMode int

const (
	// WriteModeOverwrite overwrites the target path if it already exists.
	WriteModeOverwrite WriteMode = 1 << iota

	// WriteModeCreateParents creates any missing parent directories.
	WriteModeCreateParents
)

// Has reports whether m includes every flag in other.
func (m WriteMode) Has(other WriteMode) bool {
	return m&other == other
}

// FileInfo describes a single entry returned by StateFiler.Stat or
// StateFiler.ReadDir. It deliberately does not embed os.FileInfo or fs.FileInfo
// so the interface is not tied to local-filesystem semantics.
type FileInfo interface {
	// Name returns the base name of the entry.
	Name() string

	// Size returns the size in bytes (0 for directories).
	Size() int64

	// ModTime returns the last-modified time.
	ModTime() time.Time

	// IsDir reports whether the entry is a directory.
	IsDir() bool
}

// StateFiler is the minimal file-system surface UCM state operations need.
// It is deliberately smaller than libs/filer.Filer — no Mkdir, no recursive
// delete — because the state manager and terraform wrapper never use them.
type StateFiler interface {
	// Read opens the file at path for reading.
	// Returns an error wrapping ErrNotFound when the path does not exist.
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Write writes r to path. Behavior is controlled by mode.
	Write(ctx context.Context, path string, r io.Reader, mode WriteMode) error

	// Delete removes the file at path. Deleting a non-existent path is not an error.
	Delete(ctx context.Context, path string) error

	// Stat returns metadata for the file at path.
	// Returns an error wrapping ErrNotFound when the path does not exist.
	Stat(ctx context.Context, path string) (FileInfo, error)

	// ReadDir lists the entries of the directory at path.
	ReadDir(ctx context.Context, path string) ([]FileInfo, error)
}
