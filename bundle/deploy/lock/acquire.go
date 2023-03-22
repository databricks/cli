package lock

import (
	"context"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/locker"
	"github.com/databricks/bricks/libs/log"
)

type acquire struct{}

func Acquire() bundle.Mutator {
	return &acquire{}
}

func (m *acquire) Name() string {
	return "lock:acquire"
}

func (m *acquire) init(b *bundle.Bundle) error {
	user := b.Config.Workspace.CurrentUser.UserName
	dir := b.Config.Workspace.StatePath.Workspace
	l, err := locker.CreateLocker(user, dir, b.WorkspaceClient())
	if err != nil {
		return err
	}

	b.Locker = l
	return nil
}

func (m *acquire) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Return early if locking is disabled.
	if !b.Config.Workspace.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil, nil
	}

	err := m.init(b)
	if err != nil {
		return nil, err
	}

	force := b.Config.Workspace.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = b.Locker.Lock(ctx, force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)
		return nil, err
	}

	return nil, nil
}
