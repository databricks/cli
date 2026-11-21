package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/dyn"
)

// PreDeployChecks is common set of mutators between "bundle plan" and "bundle deploy".
// Note, it is not run in "bundle migrate" so it must not modify the config
func PreDeployChecks(ctx context.Context, b *bundle.Bundle, isPlan bool) {
	bundle.ApplySeqContext(ctx, b,
		deploy.CheckDashboardsModifiedRemotely(isPlan),
		resourcemutator.SecretScopeFixups(),
		deploy.StatePull(),
		mutator.ValidateGitDetails(),
		statemgmt.CheckRunningResource(),
	)
}

// pipelineDeletionCascades reports whether deleting the pipeline referenced by a delete action
// also deletes its datasets (MVs, STs, Views). This is the server default (cascade) unless
// cascade_on_destroy is explicitly set to false.
//
// The value is read from the persisted state.
func pipelineDeletionCascades(b *bundle.Bundle, action deployplan.Action) (bool, error) {
	entry, ok := b.DeploymentBundle.StateDB.GetResourceEntry(action.ResourceKey)
	if !ok || len(entry.State) == 0 {
		return true, nil
	}

	var state dresources.PipelineState
	if err := json.Unmarshal(entry.State, &state); err != nil {
		return false, fmt.Errorf("parsing persisted state for %s: %w", action.ResourceKey, err)
	}
	if state.CascadeOnDestroy == nil {
		return true, nil
	}
	return *state.CascadeOnDestroy, nil
}

// checkForPreventDestroy checks if the resource has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed.
// If it does, it returns an error.
//
// For a resource still in the configuration the configured value decides. A resource that was
// removed from the configuration is protected by the value its last deployment recorded in the
// state, which the plan carries; the configuration cannot be consulted for it anymore.
func checkForPreventDestroy(b *bundle.Bundle, actions []deployplan.Action) error {
	root := b.Config.Value()
	var errs []error
	for _, action := range actions {
		if action.ActionType != deployplan.Recreate && action.ActionType != deployplan.Delete {
			continue
		}

		path, err := dyn.NewPathFromString(action.ResourceKey)
		if err != nil {
			return fmt.Errorf("failed to parse %q", action.ResourceKey)
		}

		if _, err := dyn.GetByPath(root, path); err != nil {
			if action.PreventDestroy {
				errs = append(errs, fmt.Errorf("%s was removed from the configuration, but its last deployment has lifecycle.prevent_destroy set. To delete it, add it back with lifecycle.prevent_destroy: false and deploy, then remove it again", action.ResourceKey))
			}
			continue
		}

		path = append(path, dyn.Key("lifecycle"), dyn.Key("prevent_destroy"))

		preventDestroyV, err := dyn.GetByPath(root, path)
		if err != nil {
			continue
		}

		preventDestroy, ok := preventDestroyV.AsBool()
		if !ok {
			return fmt.Errorf("internal error: prevent_destroy is not a boolean for %s", action.ResourceKey)
		}
		if preventDestroy {
			errs = append(errs, fmt.Errorf("%s has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed. To avoid this error, disable lifecycle.prevent_destroy for %s", action.ResourceKey, action.ResourceKey))
		}
	}

	return errors.Join(errs...)
}
