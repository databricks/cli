package lock

import (
	"context"
	"errors"
	"io/fs"

	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// workspaceFilesystemLock implements DeploymentLock using a lock file in the
// bundle's workspace state path. Holds only the primitives it needs so the
// type doesn't pin a *bundle.Bundle; the bundle-aware wiring lives in the
// NewDeploymentLock factory.
type workspaceFilesystemLock struct {
	client    *databricks.WorkspaceClient
	user      string
	statePath string
	enabled   bool
	force     bool

	// reportPermissionError explains an FS permission error from the lock
	// path back to the user. Injected so this struct doesn't import
	// bundle/permissions.
	reportPermissionError func(ctx context.Context, path string) error

	locker *locker.Locker
	goal   Goal
}

func (l *workspaceFilesystemLock) Acquire(ctx context.Context) error {
	// Return early if locking is disabled.
	if !l.enabled {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	lk, err := locker.CreateLocker(l.user, l.statePath, l.client)
	if err != nil {
		return err
	}

	l.locker = lk

	log.Infof(ctx, "Acquiring deployment lock (force: %v)", l.force)
	err = lk.Lock(ctx, l.force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)

		// If we get a permission or "doesn't exist" error from the API this
		// indicates we either don't have permissions or the path is invalid.
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
			return l.reportPermissionError(ctx, l.statePath)
		}

		return err
	}

	return nil
}

func (l *workspaceFilesystemLock) Release(ctx context.Context, _ DeploymentStatus) error {
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
		return l.locker.Unlock(ctx, locker.AllowLockFileNotExist)
	}
	return l.locker.Unlock(ctx)
}
