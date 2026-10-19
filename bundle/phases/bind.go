package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdctx"
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

	// The caller opens the state for read; close it on every path. A no-op once closed below.
	db := &b.DeploymentBundle.StateDB
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
	}()

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
	if existingID := db.GetResourceID(resourceKey); existingID != "" {
		logdiag.LogError(ctx, direct.ErrResourceAlreadyBound{ResourceKey: resourceKey, ExistingID: existingID, NewID: opts.ResourceId})
		return
	}

	if stateDesc.IsDMS() {
		// Stamp the deployment metadata deploy stamps, so the recorded state matches the config.
		muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(db.VersionID + 1)}
		if db.DeploymentID != "" {
			muts = append(muts, metadata.AnnotateDeployment(db.DeploymentID))
		}
		bundle.ApplySeqContext(ctx, b, muts...)
		if logdiag.HasError(ctx) {
			return
		}
		// The operation buffer records each state write through the workspace client on the context.
		if !cmdctx.HasWorkspaceClient(ctx) {
			ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
		}
	}

	// Plan the resource as a bind of the existing workspace resource, compared against the config.
	b.DeploymentBundle.BindKey = resourceKey
	b.DeploymentBundle.BindID = opts.ResourceId
	plan, err := b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &b.Config)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if err := selectBindPlan(plan, resourceKey); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	// Bind changes nothing; this is what the next deploy applies to the bound resource.
	entry := plan.Plan[resourceKey]
	action := deployplan.Skip
	for _, change := range entry.Changes {
		action = deployplan.GetHigherAction(action, change.Action)
	}

	// If there are changes and auto-approve is not set, show plan and ask for confirmation
	if action != deployplan.Skip && !opts.AutoApprove {
		// Display the planned changes for the bound resource
		cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", action, resourceKey))

		// Show details of what will change
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

	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// With deployment history the bind is recorded as its own version.
	if stateDesc.IsDMS() && !startBindVersion(ctx, b, plan) {
		return
	}
	b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan, false)
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
	}
	if stateDesc.IsDMS() {
		// As in deploy, push the state even after a failure, then close the version out.
		statemgmt.PushResourcesState(ctx, b)
		if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
			logdiag.LogError(ctx, err)
		}
		return
	}
	if logdiag.HasError(ctx) {
		return
	}

	statemgmt.PushResourcesState(ctx, b)
}

// selectBindPlan narrows the plan to the bound resource. Its dependencies stay for reference
// resolution but are not applied, so they must already be deployed. Its permissions and grants
// are left to the next deploy, like any other config change.
func selectBindPlan(plan *deployplan.Plan, resourceKey string) error {
	plan.FilterToSelected([]string{strings.TrimPrefix(resourceKey, "resources.")})
	for _, action := range plan.GetActions() {
		switch {
		case action.ResourceKey == resourceKey:
		case strings.HasPrefix(action.ResourceKey, resourceKey+"."):
			delete(plan.Plan, action.ResourceKey)
		case action.ActionType == deployplan.Create:
			return fmt.Errorf("cannot bind %s: it depends on %s, which is not deployed yet; deploy or bind it first", resourceKey, action.ResourceKey)
		default:
			plan.Plan[action.ResourceKey].Action = deployplan.Skip
		}
	}
	return nil
}

func jsonDump(ctx context.Context, v any, field string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Warnf(ctx, "Cannot marshal %s: %s", field, err)
		return "??"
	}
	return string(b)
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
