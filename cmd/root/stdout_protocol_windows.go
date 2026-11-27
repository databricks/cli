package root

import "os"

// sameFileInfo reports whether a and b describe the same file. On Windows os.SameFile reports any two pipes or
// character devices as the same file, since their file ids are zero. Only regular files are compared by identity, so
// a pipe or a null device is never taken for stdout.
func sameFileInfo(a, b os.FileInfo) bool {
	return a.Mode().IsRegular() && b.Mode().IsRegular() && os.SameFile(a, b)
}
