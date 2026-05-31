package lock

import (
	"context"
	"errors"
	"io/fs"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// workspaceFilesystemLock implements DeploymentManager using a lock file in the
// bundle's workspace state path. Holds only the primitives it needs from the
// bundle.
type workspaceFilesystemLock struct {
	client    *databricks.WorkspaceClient
	user      string
	statePath string
	enabled   bool
	force     bool

	// reportPermissionError produces the user-facing permission diagnostic
	// when the workspace API returns ErrPermission/ErrNotExist from Lock.
	// Lifted to a callback so this struct does not pin a *bundle.Bundle.
	reportPermissionError func(ctx context.Context, path string) diag.Diagnostics

	locker *locker.Locker
	goal   Goal
}

func (l *workspaceFilesystemLock) CreateVersion(ctx context.Context, goal Goal) (int64, error) {
	l.goal = goal

	// Return early if locking is disabled.
	if !l.enabled {
		log.Infof(ctx, "Skipping; locking is disabled")
		return 0, nil
	}

	lk, err := locker.CreateLocker(l.user, l.statePath, l.client)
	if err != nil {
		return 0, err
	}

	l.locker = lk

	log.Infof(ctx, "Acquiring deployment lock (force: %v)", l.force)
	err = lk.Lock(ctx, l.force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)

		// If we get a permission or "doesn't exist" error from the API this
		// indicates we either don't have permissions or the path is invalid.
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
			return 0, l.reportPermissionError(ctx, l.statePath).Error()
		}

		return 0, err
	}

	return 0, nil
}

func (l *workspaceFilesystemLock) CloseVersion(ctx context.Context, _ int64, _ DeploymentStatus) error {
	// Return early if locking is disabled.
	if !l.enabled {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	// Return early if the locker is not set.
	// It is likely an error occurred prior to initialization of the locker instance.
	if l.locker == nil {
		log.Warnf(ctx, "Unable to release lock if locker is not configured")
		return nil
	}

	log.Infof(ctx, "Releasing deployment lock")
	if l.goal == GoalDestroy {
		// AllowLockFileNotExist because the destroy phase deletes the remote
		// state directory, which includes the lock file itself.
		return l.locker.Unlock(ctx, locker.AllowLockFileNotExist)
	}
	return l.locker.Unlock(ctx)
}
