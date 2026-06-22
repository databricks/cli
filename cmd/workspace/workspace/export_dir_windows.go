//go:build windows

package workspace

import (
	"errors"
	"strings"
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

// reservedNameChars are the characters that are illegal in a Windows filename.
// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
const reservedNameChars = `<>:"/\|?*`

// sanitizeLocalName replaces characters that are illegal in a Windows filename
// (the reserved set plus ASCII control characters) with '_' so an object whose
// name is invalid locally can still be written under a legal name.
func sanitizeLocalName(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(reservedNameChars, r) {
			return '_'
		}
		return r
	}, name)
}
