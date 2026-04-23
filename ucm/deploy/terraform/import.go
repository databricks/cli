package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
)

// Import acquires the U2 deployment lock and runs `terraform import <address>
// <id>` in the working directory. Init is called first so main.tf.json is
// current and the provider is installed — import needs the resource block to
// exist in config before it can attach the id to it.
//
// The lock is released on defer with GoalDeploy. State mutations made by
// terraform import land in <workingDir>/terraform.tfstate; the caller is
// responsible for pushing that state via deploy.Push afterwards.
func (t *Terraform) Import(ctx context.Context, u *ucm.Ucm, address, id string) error {
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
		if relErr := locker.Release(ctx, lock.GoalDeploy); relErr != nil {
			log.Warnf(ctx, "terraform import: release lock: %v", relErr)
		}
	}()

	if err := t.runner.Import(ctx, address, id); err != nil {
		return fmt.Errorf("terraform import %s %s: %w", address, id, err)
	}
	log.Infof(ctx, "terraform import %s %s completed", address, id)
	return nil
}
