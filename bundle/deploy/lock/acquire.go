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
	err := m.init(b)
	if err != nil {
		return nil, err
	}

	force := b.Config.Bundle.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = b.Locker.Lock(ctx, force)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
