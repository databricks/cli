//go:build !windows

package lakebox

import (
	"os"
	"syscall"
)

// execSyscall replaces the current process with the given command (Unix only).
func execSyscall(path string, args []string) error {
	return syscall.Exec(path, args, os.Environ())
}
