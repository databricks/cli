package cmdio

import (
	"os"

	"golang.org/x/sys/unix"
)

// discardInput discards what was typed at the terminal f but not read yet
// (tcflush with TCIFLUSH).
func discardInput(f *os.File) error {
	return unix.IoctlSetInt(int(f.Fd()), unix.TCFLSH, unix.TCIFLUSH)
}
