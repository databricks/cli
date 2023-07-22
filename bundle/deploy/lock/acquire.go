package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
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
	dir := b.Config.Workspace.StatePath
	l, err := locker.CreateLocker(user, dir, b.WorkspaceClient())
	if err != nil {
		return err
	}

	b.Locker = l
	return nil
}

func (m *acquire) Apply(ctx context.Context, b *bundle.Bundle) error {
	// Return early if locking is disabled.
	if !b.Config.Bundle.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	err := m.init(b)
	if err != nil {
		return err
	}

	force := b.Config.Bundle.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = b.Locker.Lock(ctx, force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)
		return err
	}

	return nil
}
