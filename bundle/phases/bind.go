package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
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
		// Direct engine: import into temp state, run plan, check for changes
		// This follows the same pattern as terraform import
		groupName := terraform.TerraformToGroupName[opts.ResourceType]
		resourceKey := fmt.Sprintf("resources.%s.%s", groupName, opts.ResourceKey)
		_, statePath := b.StateFilenameDirect(ctx)

		result, err := b.DeploymentBundle.Bind(ctx, b.WorkspaceClient(), &b.Config, statePath, resourceKey, opts.ResourceId)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}

		// If there are changes and auto-approve is not set, show plan and ask for confirmation
		if result.HasChanges && !opts.AutoApprove {
			// Display the planned changes for the bound resource
			cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", result.Action, resourceKey))

			// Show details of what will change
			if result.Plan != nil {
				if entry, ok := result.Plan.Plan[resourceKey]; ok && entry != nil && len(entry.Changes) > 0 {
					cmdio.LogString(ctx, "\nChanges detected:")
					for field, change := range entry.Changes {
						if change.Action != deployplan.Skip {
							cmdio.LogString(ctx, fmt.Sprintf("  ~ %s: %v -> %v", field, jsonDump(change.Remote), jsonDump(change.New)))
						}
					}
					cmdio.LogString(ctx, "")
				}
			}

			if !cmdio.IsPromptSupported(ctx) {
				direct.CancelBind(result)
				logdiag.LogError(ctx, errors.New("This bind operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed.")) //nolint
				return
			}

			ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
			if err != nil {
				direct.CancelBind(result)
				logdiag.LogError(ctx, err)
				return
			}
			if !ans {
				direct.CancelBind(result)
				logdiag.LogError(ctx, errors.New("import aborted"))
				return
			}
		}

		// Finalize: rename temp state to final location
		err = direct.FinalizeBind(result)
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

func jsonDump(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("value=%v marshall error=%s", v, err)
	}
	return string(b)
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
