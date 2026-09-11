package filer

import (
	"io/fs"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileAlreadyExistsError_Is(t *testing.T) {
	err := fileAlreadyExistsError{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrExist)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestFileAlreadyExistsError_Error(t *testing.T) {
	err := fileAlreadyExistsError{path: "/test/path"}
	assert.Equal(t, "file already exists: /test/path", err.Error())
}

func TestFileDoesNotExistError_Is(t *testing.T) {
	err := fileDoesNotExistError{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NotErrorIs(t, err, fs.ErrExist)
}

func TestFileDoesNotExistError_Error(t *testing.T) {
	err := fileDoesNotExistError{path: "/test/path"}
	assert.Equal(t, "file does not exist: /test/path", err.Error())
}

func TestNoSuchDirectoryError_Is(t *testing.T) {
	err := noSuchDirectoryError{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NotErrorIs(t, err, fs.ErrExist)
}

func TestNoSuchDirectoryError_Error(t *testing.T) {
	err := noSuchDirectoryError{path: "/test/path"}
	assert.Equal(t, "no such directory: /test/path", err.Error())
}

func TestNotADirectory_Is(t *testing.T) {
	err := notADirectory{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrInvalid)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestNotADirectory_Error(t *testing.T) {
	err := notADirectory{path: "/test/path"}
	assert.Equal(t, "not a directory: /test/path", err.Error())
}

func TestNotAFile_Is(t *testing.T) {
	err := notAFile{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrInvalid)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestNotAFile_Error(t *testing.T) {
	err := notAFile{path: "/test/path"}
	assert.Equal(t, "not a file: /test/path", err.Error())
}

func TestDirectoryNotEmptyError_Is(t *testing.T) {
	err := directoryNotEmptyError{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrInvalid)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestDirectoryNotEmptyError_Error(t *testing.T) {
	err := directoryNotEmptyError{path: "/test/path"}
	assert.Equal(t, "directory not empty: /test/path", err.Error())
}

func TestCannotDeleteRootError_Is(t *testing.T) {
	err := cannotDeleteRootError{}
	assert.ErrorIs(t, err, fs.ErrInvalid)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestCannotDeleteRootError_Error(t *testing.T) {
	err := cannotDeleteRootError{}
	assert.Equal(t, "unable to delete filer root", err.Error())
}

func TestPermissionError_Is(t *testing.T) {
	err := permissionError{path: "/test/path"}
	assert.ErrorIs(t, err, fs.ErrPermission)
	assert.NotErrorIs(t, err, fs.ErrNotExist)
}

func TestPermissionError_Error(t *testing.T) {
	err := permissionError{path: "/test/path"}
	assert.Equal(t, "access denied: /test/path", err.Error())
}

func TestPermissionError_Unwrap(t *testing.T) {
	// A wrapped API error must remain matchable as fs.ErrPermission while also
	// being reachable via errors.As so callers can inspect its error_code.
	apiErr := &apierr.APIError{StatusCode: 403, ErrorCode: "MAX_CHILD_NODE_SIZE_EXCEEDED"}
	err := permissionError{path: "/test/path", err: apiErr}

	assert.ErrorIs(t, err, fs.ErrPermission)

	var got *apierr.APIError
	require.ErrorAs(t, err, &got)
	assert.Equal(t, "MAX_CHILD_NODE_SIZE_EXCEEDED", got.ErrorCode)
}
