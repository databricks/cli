package lock

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
)

type Goal string

const (
	GoalBind    = Goal("bind")
	GoalDeploy  = Goal("deploy")
	GoalDestroy = Goal("destroy")
)

type release struct {
	goal Goal
}

func Release(goal Goal) bundle.Mutator {
	return &release{goal}
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
	switch m.goal {
	case GoalDeploy:
		return b.Locker.Unlock(ctx)
	case GoalBind:
		return b.Locker.Unlock(ctx)
	case GoalDestroy:
		return b.Locker.Unlock(ctx, locker.AllowLockFileNotExist)
	default:
		return fmt.Errorf("unknown goal for lock release: %s", m.goal)
	}
}
