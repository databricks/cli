package filer

import "io/fs"

// Each error below is a fixed classification followed by the path it concerns.
// The classification is a source literal and safe to report to telemetry; the
// path is user data. Naming the literals here lets Error() and SafeString()
// derive from one string, so the message and what telemetry sees cannot drift.
//
// SafeString implements safeerr.SafeStringer: an error wrapped with %w by a
// safeerr error contributes its classification to that error's message
// template. See libs/safeerr.
const (
	msgFileAlreadyExists = "file already exists"
	msgFileDoesNotExist  = "file does not exist"
	msgNoSuchDirectory   = "no such directory"
	msgNotADirectory     = "not a directory"
	msgNotAFile          = "not a file"
	msgDirectoryNotEmpty = "directory not empty"
	msgCannotDeleteRoot  = "unable to delete filer root"
	msgPermissionDenied  = "access denied"
)

// fileAlreadyExistsError is returned when attempting to write a file at a path
// that already exists, without using the OverwriteIfExists WriteMode flag.
type fileAlreadyExistsError struct {
	path string
}

func (err fileAlreadyExistsError) Error() string {
	return msgFileAlreadyExists + ": " + err.path
}

func (err fileAlreadyExistsError) SafeString() string {
	return msgFileAlreadyExists
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
	return msgFileDoesNotExist + ": " + err.path
}

func (err fileDoesNotExistError) SafeString() string {
	return msgFileDoesNotExist
}

// noSuchDirectoryError is returned when attempting to write a file to a path
// whose parent directory does not exist (without CreateParentDirectories mode),
// or when attempting to read a directory that does not exist.
type noSuchDirectoryError struct {
	path string
}

func (err noSuchDirectoryError) Error() string {
	return msgNoSuchDirectory + ": " + err.path
}

func (err noSuchDirectoryError) SafeString() string {
	return msgNoSuchDirectory
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
	return msgNotADirectory + ": " + err.path
}

func (err notADirectory) SafeString() string {
	return msgNotADirectory
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
	return msgNotAFile + ": " + err.path
}

func (err notAFile) SafeString() string {
	return msgNotAFile
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
	return msgDirectoryNotEmpty + ": " + err.path
}

func (err directoryNotEmptyError) SafeString() string {
	return msgDirectoryNotEmpty
}

func (err directoryNotEmptyError) Is(other error) bool {
	return other == fs.ErrInvalid
}

// cannotDeleteRootError is returned when attempting to delete the root path
// of the filer. Deleting the root is not allowed as it would break subsequent
// file operations.
type cannotDeleteRootError struct{}

func (err cannotDeleteRootError) Error() string {
	return msgCannotDeleteRoot
}

// SafeString is the whole message: this error carries no path.
func (err cannotDeleteRootError) SafeString() string {
	return msgCannotDeleteRoot
}

func (err cannotDeleteRootError) Is(other error) bool {
	return other == fs.ErrInvalid
}

// permissionError is returned when access is denied to a path, for example
// when attempting to create a directory but lacking write permissions.
type permissionError struct {
	path string
	// err is the underlying API error, preserved so callers can inspect its
	// error_code (e.g. to distinguish a real permission denial from a workspace
	// directory that is at its child-node limit, which the API also reports as 403).
	err error
}

func (err permissionError) Error() string {
	return msgPermissionDenied + ": " + err.path
}

func (err permissionError) SafeString() string {
	return msgPermissionDenied
}

func (err permissionError) Is(other error) bool {
	return other == fs.ErrPermission
}

func (err permissionError) Unwrap() error {
	return err.err
}
