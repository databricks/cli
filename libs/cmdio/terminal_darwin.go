package cmdio

import (
	"os"

	"golang.org/x/sys/unix"
)

// discardInput discards what was typed at the terminal f but not read yet.
// It is tcflush with TCIFLUSH, which libc implements as TIOCFLUSH with FREAD.
func discardInput(f *os.File) error {
	const fread = 1
	return unix.IoctlSetPointerInt(int(f.Fd()), unix.TIOCFLUSH, fread)
}
