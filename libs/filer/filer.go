package filer

import (
	"context"
	"io"
	"io/fs"
)

// WriteMode captures intent when writing a file.
//
// The first 9 bits are reserved for the [fs.FileMode] permission bits.
// These are used only by the local filer implementation and have
// no effect for the other implementations.
type WriteMode int

// writeModePerm is a mask to extract permission bits from a WriteMode.
const writeModePerm = WriteMode(fs.ModePerm)

const (
	// Note: these constants are defined as powers of 2 to support combining them using a bit-wise OR.
	// They starts from the 10th bit (permission mask + 1) to avoid conflicts with the permission bits.
	OverwriteIfExists WriteMode = (writeModePerm + 1) << iota
	CreateParentDirectories
)

// DeleteMode captures intent when deleting a file.
type DeleteMode int

const (
	DeleteRecursively DeleteMode = 1 << iota
)

// Filer is used to access files in a workspace.
// It has implementations for accessing files in WSFS and in DBFS.
type Filer interface {
	// Write file at `path`.
	// Use the mode to further specify behavior.
	Write(ctx context.Context, path string, reader io.Reader, mode ...WriteMode) error

	// Read file at `path`.
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete file or directory at `path`.
	Delete(ctx context.Context, path string, mode ...DeleteMode) error

	// Return contents of directory at `path`.
	ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error)

	// Creates directory at `path`, creating any intermediate directories as required.
	Mkdir(ctx context.Context, path string) error

	// Stat returns information about the file at `path`.
	Stat(ctx context.Context, name string) (fs.FileInfo, error)
}
