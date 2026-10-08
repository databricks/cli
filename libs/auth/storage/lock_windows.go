package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockRegionLength is the number of bytes LockFileEx covers. The lock file has
// no contents, so locking a single byte is enough to make holders exclusive.
const lockRegionLength = 1

// tryLock takes an exclusive lock on f without blocking. It reports whether the
// lock was taken.
func tryLock(f *os.File) (bool, error) {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockRegionLength,
		0,
		&overlapped,
	)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION):
		return false, nil
	default:
		return false, err
	}
}

// unlock releases the lock held on f.
func unlock(f *os.File) {
	var overlapped windows.Overlapped
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRegionLength, 0, &overlapped)
}
