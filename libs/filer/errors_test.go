package filer

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/safeerr"
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

// TestErrorSafeStringOmitsPath is the property that lets these errors be
// reported to telemetry: the classification survives, the path does not.
func TestErrorSafeStringOmitsPath(t *testing.T) {
	const path = "/Workspace/Users/someone@example.com/secret_project"

	tests := []struct {
		err            error
		wantSafeString string
	}{
		{
			err:            fileAlreadyExistsError{path: path},
			wantSafeString: "file already exists",
		},
		{
			err:            fileDoesNotExistError{path: path},
			wantSafeString: "file does not exist",
		},
		{
			err:            noSuchDirectoryError{path: path},
			wantSafeString: "no such directory",
		},
		{
			err:            notADirectory{path: path},
			wantSafeString: "not a directory",
		},
		{
			err:            notAFile{path: path},
			wantSafeString: "not a file",
		},
		{
			err:            directoryNotEmptyError{path: path},
			wantSafeString: "directory not empty",
		},
		{
			err:            permissionError{path: path},
			wantSafeString: "access denied",
		},
		{
			err:            cannotDeleteRootError{},
			wantSafeString: "unable to delete filer root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantSafeString, func(t *testing.T) {
			safe, ok := tt.err.(interface{ SafeString() string })
			require.True(t, ok, "%T must supply a stand-in", tt.err)

			assert.Equal(t, tt.wantSafeString, safe.SafeString())
			assert.NotContains(t, safe.SafeString(), path)

			// The message still leads with the same classification.
			assert.True(t, strings.HasPrefix(tt.err.Error(), tt.wantSafeString),
				"%q should start with %q", tt.err.Error(), tt.wantSafeString)
		})
	}
}

// TestErrorSafeStringReachesTemplate covers the end-to-end path: a filer error
// wrapped by safeerr contributes its classification to the template.
func TestErrorSafeStringReachesTemplate(t *testing.T) {
	const path = "/Workspace/Users/someone@example.com/x"
	err := safeerr.Errorf("pushing direct state to workspace: %w", permissionError{path: path})

	assert.Equal(t, "pushing direct state to workspace: access denied: "+path, err.Error())
	assert.Equal(t, "pushing direct state to workspace: access denied", safeerr.SafeError(err))
	assert.NotContains(t, safeerr.SafeError(err), path)
}
