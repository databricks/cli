package filer

import "io/fs"

// fileAlreadyExistsError is returned when attempting to write a file at a path
// that already exists, without using the OverwriteIfExists WriteMode flag.
type fileAlreadyExistsError struct {
	path string
}

func (err fileAlreadyExistsError) Error() string {
	return "file already exists: " + err.path
}

func (err fileAlreadyExistsError) Is(other error) bool {
	return other == fs.ErrExist
}

// fileDoesNotExistError is returned when attempting to read, delete, or stat
// a file that does not exist. It is also returned by the workspace files
// extensions client when a notebook exists at a path but a file was expected.
type fileDoesNotExistError struct {
	path string
}

func (err fileDoesNotExistError) Is(other error) bool {
	return other == fs.ErrNotExist
}

func (err fileDoesNotExistError) Error() string {
	return "file does not exist: " + err.path
}

// noSuchDirectoryError is returned when attempting to write a file to a path
// whose parent directory does not exist (without CreateParentDirectories mode),
// or when attempting to read a directory that does not exist.
type noSuchDirectoryError struct {
	path string
}

func (err noSuchDirectoryError) Error() string {
	return "no such directory: " + err.path
}

func (err noSuchDirectoryError) Is(other error) bool {
	return other == fs.ErrNotExist
}

// notADirectory is returned when attempting to read a directory (ReadDir)
// but the path points to a file instead of a directory.
type notADirectory struct {
	path string
}

func (err notADirectory) Error() string {
	return "not a directory: " + err.path
}

func (err notADirectory) Is(other error) bool {
	return other == fs.ErrInvalid
}

// notAFile is returned when attempting to read a file but the path points
// to a directory instead of a file.
type notAFile struct {
	path string
}

func (err notAFile) Error() string {
	return "not a file: " + err.path
}

func (err notAFile) Is(other error) bool {
	return other == fs.ErrInvalid
}

// directoryNotEmptyError is returned when attempting to delete a directory
// that contains files or subdirectories, without using the DeleteRecursively
// DeleteMode flag.
type directoryNotEmptyError struct {
	path string
}

func (err directoryNotEmptyError) Error() string {
	return "directory not empty: " + err.path
}

func (err directoryNotEmptyError) Is(other error) bool {
	return other == fs.ErrInvalid
}

// cannotDeleteRootError is returned when attempting to delete the root path
// of the filer. Deleting the root is not allowed as it would break subsequent
// file operations.
type cannotDeleteRootError struct{}

func (err cannotDeleteRootError) Error() string {
	return "unable to delete filer root"
}

func (err cannotDeleteRootError) Is(other error) bool {
	return other == fs.ErrInvalid
}

// permissionError is returned when access is denied to a path, for example
// when attempting to create a directory but lacking write permissions.
type permissionError struct {
	path string
}

func (err permissionError) Error() string {
	return "access denied: " + err.path
}

func (err permissionError) Is(other error) bool {
	return other == fs.ErrPermission
}
