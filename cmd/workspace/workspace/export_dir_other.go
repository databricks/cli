//go:build !windows

package workspace

// isInvalidLocalNameError reports whether err means the workspace object could
// not be written because its name is not a legal filename on the local OS. On
// non-Windows platforms the only bytes illegal in a filename are '/' and NUL,
// neither of which can appear in a workspace object name, so this never fires.
func isInvalidLocalNameError(err error) bool {
	return false
}
