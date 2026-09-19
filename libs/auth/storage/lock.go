package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/env"
)

const (
	// tokenStoreLockFilePath is the path of the lock file guarding token store
	// updates, relative to the user's home directory. It sits next to
	// token-cache.json and is used by the keyring backend too, which has no
	// file of its own to lock.
	tokenStoreLockFilePath = ".databricks/token-cache.lock"

	// lockRetryInterval is how long to wait before retrying a contended lock.
	// The critical section is a single OAuth token exchange, so a short poll
	// keeps the wait close to the holder's actual runtime.
	lockRetryInterval = 20 * time.Millisecond
)

// LockTokenStore acquires the cross-process lock that serializes token store
// refreshes, and returns a function that releases it.
//
// The CLI is stateless, so two invocations for the same profile otherwise load
// the same cached refresh token, both exchange it, and race to write the
// result back. The lock is advisory and held only around the read-refresh-write
// sequence.
//
// It blocks until the lock is available or ctx is done. There is no timeout: the
// operating system releases the lock when a holder exits, including on a crash,
// so a lock cannot be left behind by a dead process.
func LockTokenStore(ctx context.Context) (func(), error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed loading home directory: %w", err)
	}
	path := filepath.Join(home, tokenStoreLockFilePath)
	if err := os.MkdirAll(filepath.Dir(path), ownerExecReadWrite); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, ownerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	for {
		locked, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		if locked {
			return func() {
				// Closing the file releases the lock on both platforms; the
				// explicit unlock keeps the two operations from drifting apart.
				unlock(f)
				f.Close()
			}, nil
		}

		timer := time.NewTimer(lockRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
