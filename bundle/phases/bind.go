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

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions, engine engine.EngineType) {
	log.Info(ctx, "Phase: bind")

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalBind))
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	if engine.IsDirect() {
		// Direct engine: import into temp state, run plan, check for changes
		// This follows the same pattern as terraform import
		groupName, ok := terraform.TerraformToGroupName[opts.ResourceType]
		if !ok {
			groupName = opts.ResourceType
		}
		resourceKey := fmt.Sprintf("resources.%s.%s", groupName, opts.ResourceKey)

		if b.ConfiguresDeploymentHistory(ctx) {
			// A recorded deployment keeps its resources in the metadata service, so the bind is
			// recorded there rather than written to the state file.
			bindWithHistory(ctx, b, resourceKey, opts.ResourceId, opts.AutoApprove)
			if logdiag.HasError(ctx) {
				return
			}
		} else {
			_, statePath := b.StateFilenameDirect(ctx)

			result, err := b.DeploymentBundle.Bind(ctx, b.WorkspaceClient(ctx), &b.Config, statePath, resourceKey, opts.ResourceId)
			if err != nil {
				logdiag.LogError(ctx, err)
				return
			}

			if !confirmBindPlan(ctx, resourceKey, result.Plan, opts.AutoApprove, false) {
				result.Cancel()
				return
			}

			// Finalize: rename temp state to final location
			if err := result.Finalize(); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
		}
	} else {
		// Terraform engine: use terraform import
		bundle.ApplySeqContext(
			ctx, b,
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

// confirmBindPlan shows the bound resource's planned action and, unless autoApprove, asks the user
// to confirm. It reports whether the bind should proceed; on decline or an unpromptable console it
// logs the reason and returns false. A plain bind or skip changes nothing, so it proceeds without a
// prompt. immediate is true when the caller applies the change now (the DMS path) rather than
// deferring it to the next deploy, so the prompt describes the right timing. The caller owns any
// cleanup on a false return.
func confirmBindPlan(ctx context.Context, resourceKey string, plan *deployplan.Plan, autoApprove, immediate bool) bool {
	var entry *deployplan.PlanEntry
	if plan != nil {
		entry = plan.Plan[resourceKey]
	}
	changesWorkspace := entry != nil && entry.Action != deployplan.Skip && entry.Action != deployplan.Bind && entry.Action != deployplan.Undefined
	if !changesWorkspace || autoApprove {
		return true
	}

	cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", entry.Action, resourceKey))
	if len(entry.Changes) > 0 {
		cmdio.LogString(ctx, "\nChanges detected:")
		for _, field := range slices.Sorted(maps.Keys(entry.Changes)) {
			change := entry.Changes[field]
			if change.Action != deployplan.Skip {
				cmdio.LogString(ctx, fmt.Sprintf("  ~ %s: %v -> %v", field, jsonDump(ctx, change.Remote, field), jsonDump(ctx, change.New, field)))
			}
		}
		cmdio.LogString(ctx, "")
	}

	if !cmdio.IsPromptSupported(ctx) {
		logdiag.LogError(ctx, fmt.Errorf("this bind operation requires user confirmation, but the current console does not support prompting.\nTo proceed, use --auto-approve after reviewing the plan above.%s", agent.AgentNotice()))
		return false
	}

	prompt := "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'."
	if immediate {
		prompt = "Confirm bind? The change will be applied to the workspace now."
	}
	ans, err := cmdio.AskYesOrNo(ctx, prompt)
	if err != nil {
		logdiag.LogError(ctx, err)
		return false
	}
	if !ans {
		logdiag.LogError(ctx, errors.New("import aborted"))
		return false
	}

	return true
}

func Unbind(ctx context.Context, b *bundle.Bundle, bundleType, tfResourceType, resourceKey string, engine engine.EngineType) {
	log.Info(ctx, "Phase: unbind")

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalUnbind))
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	if engine.IsDirect() {
		groupName, ok := terraform.TerraformToGroupName[tfResourceType]
		if !ok {
			groupName = tfResourceType
		}
		fullResourceKey := fmt.Sprintf("resources.%s.%s", groupName, resourceKey)
		// Unbind under the deployment metadata service is not supported yet (no unbind operation
		// action type exists); DeploymentBundle.Unbind errors for a recorded deployment.
		_, statePath := b.StateFilenameDirect(ctx)
		if err := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		bundle.ApplySeqContext(
			ctx, b,
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
