package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
)

// StateRm acquires the U2 deployment lock and runs `terraform state rm
// <address>` in the working directory. Init is called first to ensure the
// provider is installed and main.tf.json reflects the current config — state
// rm does not read the config but init is how we bootstrap the runner.
//
// The lock is released on defer with GoalUnbind so the recorded goal matches
// the caller's intent. State mutations land in <workingDir>/terraform.tfstate;
// the caller is responsible for pushing that state via deploy.Push afterwards.
func (t *Terraform) StateRm(ctx context.Context, u *ucm.Ucm, address string) error {
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
	if err := locker.Acquire(ctx, false); err != nil {
		return err
	}
	defer func() {
		if relErr := locker.Release(ctx, lock.GoalUnbind); relErr != nil {
			log.Warnf(ctx, "terraform state rm: release lock: %v", relErr)
		}
	}()

	if err := t.runner.StateRm(ctx, address); err != nil {
		return fmt.Errorf("terraform state rm %s: %w", address, err)
	}
	log.Infof(ctx, "terraform state rm %s completed", address)
	return nil
}
