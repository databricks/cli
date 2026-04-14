package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
)

type workspaceFilesystemLock struct {
	b    *bundle.Bundle
	goal Goal
}

func newWorkspaceFilesystemLock(b *bundle.Bundle, goal Goal) *workspaceFilesystemLock {
	return &workspaceFilesystemLock{b: b, goal: goal}
}

func (l *workspaceFilesystemLock) Acquire(ctx context.Context) error {
	b := l.b

	if !b.Config.Bundle.Deployment.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	user := b.Config.Workspace.CurrentUser.UserName
	dir := b.Config.Workspace.StatePath
	lk, err := locker.CreateLocker(user, dir, b.WorkspaceClient())
	if err != nil {
		return err
	}

	b.Locker = lk

	force := b.Config.Bundle.Deployment.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = lk.Lock(ctx, force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)
		return err
	}

	return nil
}

func (l *workspaceFilesystemLock) Release(ctx context.Context, _ DeploymentStatus) error {
	b := l.b

	if !b.Config.Bundle.Deployment.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

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
