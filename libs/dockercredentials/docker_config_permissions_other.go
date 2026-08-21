//go:build !windows

package dockercredentials

import "os"

// createOwnerOnlyTempFile relies on os.CreateTemp's 0600 mode outside Windows.
func createOwnerOnlyTempFile(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}
