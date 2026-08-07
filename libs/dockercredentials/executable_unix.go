//go:build !windows

package dockercredentials

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func isExecutableForPath(path string, mode os.FileMode, goos string) bool {
	if goos == "windows" {
		return true
	}

	err := unix.Faccessat(unix.AT_FDCWD, path, unix.X_OK, unix.AT_EACCESS)
	if err == nil {
		return true
	}
	if errors.Is(err, unix.ENOSYS) {
		return mode&0o111 != 0
	}
	return false
}
