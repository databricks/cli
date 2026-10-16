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
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

type BindOptions struct {
	AutoApprove bool
	// Group is the bundle resource group, e.g. "jobs".
	Group       string
	ResourceKey string
	ResourceId  string
}

func Bind(ctx context.Context, b *bundle.Bundle, opts *BindOptions, stateDesc *statemgmt.StateDesc) {
	log.Info(ctx, "Phase: bind")
	engine := stateDesc.Engine

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalBind))
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	// The terraform engine was removed in v1.20.0. bind does not migrate on its own, so a
	// terraform state means the user must migrate first (via "bundle deploy").
	if !engine.IsDirect() {
		logdiag.LogError(ctx, errors.New(TerraformStateRemovedMessage))
		return
	}

	resourceKey := fmt.Sprintf("resources.%s.%s", opts.Group, opts.ResourceKey)
	if stateDesc.IsDMS() {
		bindWithHistory(ctx, b, resourceKey, opts.ResourceId, opts.AutoApprove)
		return
	}

	// Direct engine: import into temp state, run plan, check for changes.
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
	err = result.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	statemgmt.PushResourcesState(ctx, b)
}

func jsonDump(ctx context.Context, v any, field string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Warnf(ctx, "Cannot marshal %s: %s", field, err)
		return "??"
	}
	return string(b)
}

// confirmBindPlan shows the planned changes and, unless autoApprove, asks the user to approve them.
// It reports whether the bind should proceed. An immediate bind (DMS) also applies the bound
// resource's permissions and grants now, so their changes need approval too.
func confirmBindPlan(ctx context.Context, resourceKey string, plan *deployplan.Plan, autoApprove, immediate bool) bool {
	if autoApprove {
		return true
	}
	changesWorkspace := false
	for _, action := range plan.GetActions() {
		if !immediate && action.ResourceKey != resourceKey {
			continue
		}
		if action.ActionType == deployplan.Skip || action.ActionType == deployplan.Bind || action.ActionType == deployplan.Undefined {
			continue
		}
		changesWorkspace = true
		entry := plan.Plan[action.ResourceKey]
		cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", entry.Action, action.ResourceKey))
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
	}
	if !changesWorkspace {
		return true
	}

	if !cmdio.IsPromptSupported(ctx) {
		logdiag.LogError(ctx, fmt.Errorf("this bind operation requires user confirmation, but the current console does not support prompting.\nTo proceed, use --auto-approve after reviewing the plan above.%s", agent.AgentNotice()))
		return false
	}

	prompt := "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'."
	if immediate {
		prompt = "Confirm bind? The changes will be applied to the workspace now."
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

func Unbind(ctx context.Context, b *bundle.Bundle, group, resourceKey string, engine engine.EngineType) {
	log.Info(ctx, "Phase: unbind")

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalUnbind))
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	// The terraform engine was removed in v1.20.0. unbind does not migrate on its own, so a
	// terraform state means the user must migrate first (via "bundle deploy").
	if !engine.IsDirect() {
		logdiag.LogError(ctx, errors.New(TerraformStateRemovedMessage))
		return
	}

	fullResourceKey := fmt.Sprintf("resources.%s.%s", group, resourceKey)
	_, statePath := b.StateFilenameDirect(ctx)
	err := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	statemgmt.PushResourcesState(ctx, b)
}
