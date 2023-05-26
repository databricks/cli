package filer

import (
	"context"
	"fmt"
	"io"
	"time"
)

type WriteMode int

const (
	OverwriteIfExists       WriteMode = iota
	CreateParentDirectories           = iota << 1
)

type FileInfo struct {
	// The type of the file in workspace
	Type string

	// Base name of the file
	Name string

	// Size in bytes of the file
	Size int64

	// Last Modified time of the file
	ModTime time.Time
}

type FileAlreadyExistsError struct {
	path string
}

func (err FileAlreadyExistsError) Error() string {
	return fmt.Sprintf("file already exists: %s", err.path)
}

type NoSuchDirectoryError struct {
	path string
}

func (err NoSuchDirectoryError) Error() string {
	return fmt.Sprintf("no such directory: %s", err.path)
}

// Filer is used to access files in a workspace.
// It has implementations for accessing files in WSFS and in DBFS.
type Filer interface {
	// Write file at `path`.
	// Use the mode to further specify behavior.
	Write(ctx context.Context, path string, reader io.Reader, mode ...WriteMode) error

	// Read file at `path`.
	Read(ctx context.Context, path string) (io.Reader, error)

	// Delete file at `path`.
	Delete(ctx context.Context, path string) error

	// Return contents of directory at `path`
	ReadDir(ctx context.Context, path string) ([]FileInfo, error)
}
