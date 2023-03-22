package lock

import (
	"context"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/log"
)

type release struct{}

func Release() bundle.Mutator {
	return &release{}
}

func (m *release) Name() string {
	return "lock:release"
}

func (m *release) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Return early if the locker is not set.
	// It is likely an error occurred prior to initialization of the locker instance.
	if b.Locker == nil {
		log.Warnf(ctx, "Unable to release lock if locker is not configured")
		return nil, nil
	}

	log.Infof(ctx, "Releasing deployment lock")
	err := b.Locker.Unlock(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
