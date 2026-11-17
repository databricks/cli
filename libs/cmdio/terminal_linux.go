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

// discardInput discards what was typed at the terminal f but not read yet
// (tcflush with TCIFLUSH).
func discardInput(f *os.File) error {
	return unix.IoctlSetInt(int(f.Fd()), unix.TCFLSH, unix.TCIFLUSH)
}
