package lock

import (
	"context"
	"errors"
	"io/fs"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
)

// workspaceFilesystemLock implements DeploymentLock using a lock file in the
// bundle's workspace state path. This preserves the historical behavior of
// the previous lock.Acquire / lock.Release mutators.
type workspaceFilesystemLock struct {
	b    *bundle.Bundle
	goal Goal
}

func newWorkspaceFilesystemLock(b *bundle.Bundle, goal Goal) *workspaceFilesystemLock {
	return &workspaceFilesystemLock{b: b, goal: goal}
}

func (l *workspaceFilesystemLock) Acquire(ctx context.Context) error {
	b := l.b

	// Return early if locking is disabled.
	if !b.Config.Bundle.Deployment.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	user := b.Config.Workspace.CurrentUser.UserName
	dir := b.Config.Workspace.StatePath
	lk, err := locker.CreateLocker(user, dir, b.WorkspaceClient(ctx))
	if err != nil {
		return err
	}

	b.Locker = lk

	force := b.Config.Bundle.Deployment.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = lk.Lock(ctx, force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)

		// If we get a permission or "doesn't exist" error from the API this
		// indicates we either don't have permissions or the path is invalid.
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
			diags := permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.StatePath)
			return diags.Error()
		}

		return err
	}

	return nil
}

func (l *workspaceFilesystemLock) Release(ctx context.Context, _ DeploymentStatus) error {
	b := l.b

	// Return early if locking is disabled.
	if !b.Config.Bundle.Deployment.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	// Return early if the locker is not set.
	// It is likely an error occurred prior to initialization of the locker instance.
	if b.Locker == nil {
		log.Warnf(ctx, "Unable to release lock if locker is not configured")
		return nil
	}

	log.Infof(ctx, "Releasing deployment lock")
	if l.goal == GoalDestroy {
		return b.Locker.Unlock(ctx, locker.AllowLockFileNotExist)
	}
	return b.Locker.Unlock(ctx)
}
