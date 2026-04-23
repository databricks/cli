package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
)

// Destroy acquires the U2 deployment lock and runs `terraform destroy
// -auto-approve` in the working directory. Init is called first to make
// sure main.tf.json is up to date (destroy still needs the config to know
// the resource graph) and the provider is installed.
//
// The lock is released on defer with GoalDestroy, which tolerates a
// missing lock file (destroy may have wiped the state dir before Release
// runs — see ucm/deploy/lock.Release).
func (t *Terraform) Destroy(ctx context.Context, u *ucm.Ucm, forceLock bool) error {
	if t == nil {
		return fmt.Errorf("terraform: nil wrapper")
	}

	if err := t.Init(ctx, u); err != nil {
		return err
	}

	factory := t.lockerFactory
	if factory == nil {
		factory = defaultLockerFactory
	}
	locker, err := factory(ctx, u, t.user)
	if err != nil {
		return fmt.Errorf("create deployment locker: %w", err)
	}
	if err := locker.Acquire(ctx, forceLock); err != nil {
		return err
	}
	defer func() {
		if relErr := locker.Release(ctx, lock.GoalDestroy); relErr != nil {
			log.Warnf(ctx, "terraform destroy: release lock: %v", relErr)
		}
	}()

	if err := t.runner.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}
	log.Infof(ctx, "terraform destroy completed")
	return nil
}
