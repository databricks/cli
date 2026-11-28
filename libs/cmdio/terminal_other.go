//go:build !linux && !darwin && !windows

package cmdio

import "os"

// discardInput discards what was typed at the terminal f but not read yet.
// Only the release builds' platforms implement it.
func discardInput(*os.File) error {
	return nil
}
