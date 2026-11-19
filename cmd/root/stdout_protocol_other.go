//go:build !windows

package root

import "os"

// sameFileInfo reports whether a and b describe the same file.
func sameFileInfo(a, b os.FileInfo) bool {
	return os.SameFile(a, b)
}
