package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions, engine engine.EngineType) (err error) {
	log.Info(ctx, "Phase: bind")

	if acquireErr := bundle.ApplyContext(ctx, b, lock.Acquire()); acquireErr != nil {
		return acquireErr
	}

	defer func() {
		err = logdiag.FlushError(ctx, err)
		if releaseErr := bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind)); releaseErr != nil && err == nil {
			err = logdiag.FlushError(ctx, releaseErr)
		}
	}()

	if engine.IsDirect() {
		// Direct engine: import into temp state, run plan, check for changes
		// This follows the same pattern as terraform import
		groupName, ok := terraform.TerraformToGroupName[opts.ResourceType]
		if !ok {
			groupName = opts.ResourceType
		}
		resourceKey := fmt.Sprintf("resources.%s.%s", groupName, opts.ResourceKey)
		_, statePath := b.StateFilenameDirect(ctx)

		result, bindErr := b.DeploymentBundle.Bind(ctx, b.WorkspaceClient(ctx), &b.Config, statePath, resourceKey, opts.ResourceId)
		if bindErr != nil {
			return bindErr
		}

		// If there are changes and auto-approve is not set, show plan and ask for confirmation
		if result.HasChanges && !opts.AutoApprove {
			// Display the planned changes for the bound resource
			cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", result.Action, resourceKey))

			// Show details of what will change
			if result.Plan != nil {
				if entry, ok := result.Plan.Plan[resourceKey]; ok && entry != nil && len(entry.Changes) > 0 {
					cmdio.LogString(ctx, "\nChanges detected:")
					for _, field := range slices.Sorted(maps.Keys(entry.Changes)) {
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
				return fmt.Errorf("this bind operation requires user confirmation, but the current console does not support prompting.\nTo proceed, use --auto-approve after reviewing the plan above.%s", agent.AgentNotice())
			}

			ans, askErr := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
			if askErr != nil {
				result.Cancel()
				return askErr
			}
			if !ans {
				result.Cancel()
				return errors.New("import aborted")
			}
		}

		// Finalize: rename temp state to final location
		if finalizeErr := result.Finalize(); finalizeErr != nil {
			return finalizeErr
		}
	} else {
		// Terraform engine: use terraform import
		if seqErr := bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Import(opts),
		); seqErr != nil {
			return seqErr
		}
	}

	return statemgmt.PushResourcesState(ctx, b, engine)
}

func jsonDump(ctx context.Context, v any, field string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Warnf(ctx, "Cannot marshal %s: %s", field, err)
		return "??"
	}
	return string(b)
}

func Unbind(ctx context.Context, b *bundle.Bundle, bundleType, tfResourceType, resourceKey string, engine engine.EngineType) (err error) {
	log.Info(ctx, "Phase: unbind")

	if acquireErr := bundle.ApplyContext(ctx, b, lock.Acquire()); acquireErr != nil {
		return acquireErr
	}

	defer func() {
		err = logdiag.FlushError(ctx, err)
		if releaseErr := bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind)); releaseErr != nil && err == nil {
			err = logdiag.FlushError(ctx, releaseErr)
		}
	}()

	if engine.IsDirect() {
		groupName, ok := terraform.TerraformToGroupName[tfResourceType]
		if !ok {
			groupName = tfResourceType
		}
		fullResourceKey := fmt.Sprintf("resources.%s.%s", groupName, resourceKey)
		_, statePath := b.StateFilenameDirect(ctx)
		if unbindErr := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey); unbindErr != nil {
			return unbindErr
		}
	} else {
		if seqErr := bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Unbind(bundleType, tfResourceType, resourceKey),
		); seqErr != nil {
			return seqErr
		}
	}

	return statemgmt.PushResourcesState(ctx, b, engine)
}
