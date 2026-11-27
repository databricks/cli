//go:build !linux && !darwin && !windows

package cmdio

import "os"

// inForeground reports whether this process may read the terminal f. The
// release builds' platforms check it; others assume it may.
func inForeground(*os.File) bool {
	return true
}

// discardInput discards what was typed at the terminal f but not read yet.
// Only the release builds' platforms implement it.
func discardInput(*os.File) error {
	return nil
}
