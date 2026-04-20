package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Apply acquires the U2 deployment lock and runs `terraform apply`.
// If a plan artefact from a previous Plan call is available, Apply consumes
// it via tfexec.DirOrPlan; otherwise it falls back to an auto-approved
// `terraform apply` that computes its own plan inline.
//
// The lock is released on defer with the GoalDeploy value regardless of
// whether Apply succeeded. Contention on the lock surfaces as a
// *lock.ErrLockHeld so callers can errors.As on it and present a helpful
// "--force-lock to override" message to the user.
func (t *Terraform) Apply(ctx context.Context, u *ucm.Ucm) error {
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
			log.Warnf(ctx, "terraform apply: release lock: %v", relErr)
		}
	}()

	opts := []tfexec.ApplyOption{}
	if t.lastPlanExists && t.lastPlanPath != "" {
		opts = append(opts, tfexec.DirOrPlan(t.lastPlanPath))
	}
	if err := t.runner.Apply(ctx, opts...); err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	log.Infof(ctx, "terraform apply completed")
	return nil
}
