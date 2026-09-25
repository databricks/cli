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
	lockRetryInterval = 20 * time.Millisecond

	// lockWaitTimeout bounds the wait behind a live holder. A holder performs
	// one token exchange (bounded by the 30 second HTTP client timeout) and one
	// store write (bounded by the 3 second keyring timeout), so a longer wait
	// means the holder is stuck rather than slow.
	lockWaitTimeout = time.Minute
)

// lockTokenStore acquires the cross-process lock that serializes token store
// refreshes, and returns a function that releases it.
//
// There is a single lock for all profiles and both storage modes: the
// plaintext backend rewrites the whole token-cache.json on every Put, so
// per-profile locks would let refreshes of different profiles overwrite each
// other's entries. A dead holder cannot leave the lock behind because the
// operating system releases it when the process exits.
//
// It blocks until the lock is available, ctx is done, or lockWaitTimeout
// elapses.
func lockTokenStore(ctx context.Context) (func(), error) {
	return lockTokenStoreWithTimeout(ctx, lockWaitTimeout)
}

func lockTokenStoreWithTimeout(ctx context.Context, waitTimeout time.Duration) (func(), error) {
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

	deadline := time.NewTimer(waitTimeout)
	defer deadline.Stop()
	for {
		locked, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("lock %s: %w", filepath.ToSlash(path), err)
		}
		if locked {
			return func() {
				// Closing the file releases the lock on both platforms; the
				// explicit unlock keeps the two operations from drifting apart.
				unlock(f)
				f.Close()
			}, nil
		}

		retry := time.NewTimer(lockRetryInterval)
		select {
		case <-ctx.Done():
			retry.Stop()
			f.Close()
			return nil, ctx.Err()
		case <-deadline.C:
			retry.Stop()
			f.Close()
			return nil, fmt.Errorf("timed out after %s waiting for another databricks process to release %s", waitTimeout, filepath.ToSlash(path))
		case <-retry.C:
		}
	}
}
