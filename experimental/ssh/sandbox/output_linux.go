package sandbox

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func duplicateServerOutput(file *os.File) (io.WriteCloser, error) {
	// Command execution owns the pipe; reopening /proc/self/fd requires its
	// owner's permissions. Dup retains the inherited descriptor's access.
	fd, err := unix.FcntlInt(file.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	// NewFile registers a nonblocking pipe with Go's poller so Close can
	// interrupt a heartbeat write when command output stops being consumed.
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), "sandbox command output"), nil
}
