package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions) {
	log.Info(ctx, "Phase: bind")

	eng, err := engine.FromEnv(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	if eng.IsDirect() {

		// Note, difference from Terraform engine:
		// We simply always ask for confirmation or for --auto-approve.
		// Terraform logic is to only do that if bind results in plan changes.

		if !opts.AutoApprove {
			if !cmdio.IsPromptSupported(ctx) {
				logdiag.LogError(ctx, errors.New("This bind operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed."))
				return
			}

			ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
			if err != nil {
				logdiag.LogError(ctx, err)
				return
			}
			if !ans {
				logdiag.LogError(ctx, errors.New("import aborted"))
				return
			}
		}

		// Direct engine: simply add the resource to state
		groupName := terraform.TerraformToGroupName[opts.ResourceType]
		resourceKey := fmt.Sprintf("resources.%s.%s", groupName, opts.ResourceKey)
		_, statePath := b.StateFilenameDirect(ctx)
		err := b.DeploymentBundle.Bind(ctx, statePath, resourceKey, opts.ResourceId)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		// Terraform engine: use terraform import
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Import(opts),
		)
		if logdiag.HasError(ctx) {
			return
		}
	}

	statemgmt.PushResourcesState(ctx, b, eng)
}

func Unbind(ctx context.Context, b *bundle.Bundle, bundleType, tfResourceType, resourceKey string) {
	log.Info(ctx, "Phase: unbind")

	eng, err := engine.FromEnv(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	if eng.IsDirect() {
		// Direct engine: simply remove the resource from state
		groupName := terraform.TerraformToGroupName[tfResourceType]
		fullResourceKey := fmt.Sprintf("resources.%s.%s", groupName, resourceKey)
		_, statePath := b.StateFilenameDirect(ctx)
		err := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		// Terraform engine: use terraform state rm
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Unbind(bundleType, tfResourceType, resourceKey),
		)
		if logdiag.HasError(ctx) {
			return
		}
	}

	statemgmt.PushResourcesState(ctx, b, eng)
}
