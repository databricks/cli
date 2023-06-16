package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
)

type release struct {
	allowLockFileNotExist bool
}

func Release(allowLockFileNotExist bool) bundle.Mutator {
	return &release{allowLockFileNotExist}
}

func (m *release) Name() string {
	return "lock:release"
}

func (m *release) Apply(ctx context.Context, b *bundle.Bundle) error {
	// Return early if locking is disabled.
	if !b.Config.Bundle.Lock.IsEnabled() {
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
	err := b.Locker.Unlock(ctx, m.allowLockFileNotExist)
	if err != nil {
		log.Errorf(ctx, "Failed to release deployment lock: %v", err)
		return err
	}

	return nil
}
