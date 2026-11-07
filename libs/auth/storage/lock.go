package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// lockRetryInterval is how long to wait before retrying a contended lock.
	lockRetryInterval = 20 * time.Millisecond
)

// lockTokenStore acquires the cross-process lock that serializes token store
// mutations. The stable sidecar survives atomic replacement of the cache.
func lockTokenStore(ctx context.Context, path string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), ownerExecReadWrite); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, ownerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return nil, err
		}
		locked, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("lock %s: %w", filepath.ToSlash(path), err)
		}
		if locked {
			if err := ctx.Err(); err != nil {
				unlock(f)
				f.Close()
				return nil, err
			}
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
		case <-retry.C:
		}
	}
}
