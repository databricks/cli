//go:build linux || darwin

package cmdio

import (
	"os"

	"golang.org/x/sys/unix"
)

// inForeground reports whether this process is in the foreground process
// group of the terminal f. A background job that reads its terminal is
// stopped (SIGTTIN) until the user brings it to the foreground.
func inForeground(f *os.File) bool {
	pgrp, err := unix.IoctlGetInt(int(f.Fd()), unix.TIOCGPGRP)
	return err == nil && pgrp == unix.Getpgrp()
}
