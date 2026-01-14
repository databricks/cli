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
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/utils"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions) {
	log.Info(ctx, "Phase: bind")

	engine, err := engine.FromEnv(ctx)
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

	if engine.IsDirect() {
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
					for _, field := range utils.SortedKeys(entry.Changes) {
						change := entry.Changes[field]
						if change.Action != deployplan.Skip {
							cmdio.LogString(ctx, fmt.Sprintf("  ~ %s: %v -> %v", field, jsonDump(ctx, change.Remote, field), jsonDump(ctx, change.New, field)))
						}
					}
					cmdio.LogString(ctx, "")
				}
			}

			if !cmdio.IsPromptSupported(ctx) {
				result.Cancel()
				logdiag.LogError(ctx, errors.New("This bind operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed.")) //nolint
				return
			}

			ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
			if err != nil {
				result.Cancel()
				logdiag.LogError(ctx, err)
				return
			}
			if !ans {
				result.Cancel()
				logdiag.LogError(ctx, errors.New("import aborted"))
				return
			}
		}

		// Finalize: rename temp state to final location
		err = result.Finalize()
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

	statemgmt.PushResourcesState(ctx, b, engine)
}

func jsonDump(ctx context.Context, v any, field string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Warnf(ctx, "Cannot marshal %s: %s", field, err)
		return "??"
	}
	return string(b)
}

func Unbind(ctx context.Context, b *bundle.Bundle, bundleType, tfResourceType, resourceKey string) {
	log.Info(ctx, "Phase: unbind")

	engine, err := engine.FromEnv(ctx)
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

	if engine.IsDirect() {
		groupName := terraform.TerraformToGroupName[tfResourceType]
		fullResourceKey := fmt.Sprintf("resources.%s.%s", groupName, resourceKey)
		_, statePath := b.StateFilenameDirect(ctx)
		err := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Unbind(bundleType, tfResourceType, resourceKey),
		)
		if logdiag.HasError(ctx) {
			return
		}
	}

	statemgmt.PushResourcesState(ctx, b, engine)
}
