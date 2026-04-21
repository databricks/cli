package lock

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/databricks/cli/libs/log"
)

// ReleaseOption tweaks Release behavior.
type ReleaseOption int

const (
	// AllowLockFileNotExist makes Release succeed if the lock file is
	// already gone. Used by destroy, which may delete the state dir before
	// Release gets a chance to run.
	AllowLockFileNotExist ReleaseOption = iota
)

// Release deletes the lock file under TargetDir if this locker owns it.
// The goal argument mirrors bundle/deploy/lock.Release: destroy implies
// AllowLockFileNotExist is tolerated even without an explicit option.
func (l *Locker) Release(ctx context.Context, goal Goal, opts ...ReleaseOption) error {
	log.Infof(ctx, "Releasing deployment lock (goal: %s)", goal)

	if !l.Active {
		return errors.New("unlock called when lock is not held")
	}

	// Destroy tolerates a missing lock file by default.
	allowNotExist := slices.Contains(opts, AllowLockFileNotExist) || goal == GoalDestroy

	if _, err := l.filer.Stat(ctx, LockFileName); errors.Is(err, fs.ErrNotExist) && allowNotExist {
		l.Active = false
		return nil
	}

	if err := l.assertLockHeld(ctx); err != nil {
		return fmt.Errorf("unlock called when lock is not held: %w", err)
	}

	if err := l.filer.Delete(ctx, LockFileName); err != nil {
		return err
	}
	l.Active = false
	return nil
}
