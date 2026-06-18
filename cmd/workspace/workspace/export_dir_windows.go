//go:build windows

package workspace

import (
	"errors"
	"syscall"
)

// errorInvalidName is the Windows ERROR_INVALID_NAME code. The file APIs return
// it when a path contains characters that are illegal in a local filename, such
// as the ':' in a notebook named "New Notebook 2026-05-04 13:54:24". It is not
// declared in the standard syscall package, so we use the well-known code.
// https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499-
const errorInvalidName = syscall.Errno(0x7b)

// isInvalidLocalNameError reports whether err means the workspace object could
// not be written because its name is not a legal filename on the local OS.
func isInvalidLocalNameError(err error) bool {
	return errors.Is(err, errorInvalidName)
}
