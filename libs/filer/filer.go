package filer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
)

type WriteMode int

const (
	OverwriteIfExists       WriteMode = iota
	CreateParentDirectories           = iota << 1
)

type FileAlreadyExistsError struct {
	path string
}

func (err FileAlreadyExistsError) Error() string {
	return fmt.Sprintf("file already exists: %s", err.path)
}

func (err FileAlreadyExistsError) Is(other error) bool {
	return other == fs.ErrExist
}

type FileDoesNotExistError struct {
	path string
}

func (err FileDoesNotExistError) Is(other error) bool {
	return other == fs.ErrNotExist
}

func (err FileDoesNotExistError) Error() string {
	return fmt.Sprintf("file does not exist: %s", err.path)
}

type NoSuchDirectoryError struct {
	path string
}

func (err NoSuchDirectoryError) Error() string {
	return fmt.Sprintf("no such directory: %s", err.path)
}

func (err NoSuchDirectoryError) Is(other error) bool {
	return other == fs.ErrNotExist
}

type NotADirectory struct {
	path string
}

func (err NotADirectory) Error() string {
	return fmt.Sprintf("not a directory: %s", err.path)
}

func (err NotADirectory) Is(other error) bool {
	return other == fs.ErrInvalid
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

	// Return contents of directory at `path`.
	ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error)

	// Creates directory at `path`, creating any intermediate directories as required.
	Mkdir(ctx context.Context, path string) error

	// Stat returns information about the file at `path`.
	Stat(ctx context.Context, name string) (fs.FileInfo, error)
}
