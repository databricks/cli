// Package filelock coordinates processes through an exclusive lock held on a
// persistent file descriptor.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	pollInterval = 20 * time.Millisecond
	maxWait      = time.Minute
)

var (
	errTimeout     = errors.New("timed out waiting for file lock")
	errInvalidWait = errors.New("wait duration must be greater than zero")
)

// Acquire obtains an exclusive descriptor lock on path, creating the lock file
// with mode 0600 if it does not exist. Lock attempts are nonblocking and repeat
// every 20 milliseconds. The wait is capped at one minute, and context
// cancellation takes precedence when it occurs first.
//
// The returned release function unlocks and closes the descriptor. The lock
// file itself is persistent.
func Acquire(ctx context.Context, path string, wait time.Duration) (func() error, error) {
	if wait <= 0 {
		return nil, errInvalidWait
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file %q: %w", path, err)
	}

	if err := ctx.Err(); err != nil {
		return nil, closeWithError(file, err)
	}

	var timeout *time.Timer
	var ticker *time.Ticker

	for {
		acquired, err := tryLock(file)
		if err != nil {
			stopWaiters(timeout, ticker)
			return nil, closeWithError(file, fmt.Errorf("acquiring lock on %q: %w", path, err))
		}
		if acquired {
			stopWaiters(timeout, ticker)
			if err := ctx.Err(); err != nil {
				return nil, errors.Join(err, unlock(file), file.Close())
			}
			return func() error {
				return errors.Join(unlock(file), file.Close())
			}, nil
		}

		if timeout == nil {
			timeout = time.NewTimer(min(wait, maxWait))
		}
		if ticker == nil {
			ticker = time.NewTicker(pollInterval)
		}
		if err := waitForRetry(ctx, timeout, ticker); err != nil {
			stopWaiters(timeout, ticker)
			return nil, closeWithError(file, err)
		}
	}
}

func waitForRetry(ctx context.Context, timeout *time.Timer, ticker *time.Ticker) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout.C:
		if err := ctx.Err(); err != nil {
			return err
		}
		return errTimeout
	case <-ticker.C:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout.C:
		if err := ctx.Err(); err != nil {
			return err
		}
		return errTimeout
	default:
		return nil
	}
}

func stopWaiters(timeout *time.Timer, ticker *time.Ticker) {
	if timeout != nil {
		timeout.Stop()
	}
	if ticker != nil {
		ticker.Stop()
	}
}

func closeWithError(file *os.File, err error) error {
	return errors.Join(err, file.Close())
}
