package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/dyn"
)

// PreDeployChecks is common set of mutators between "bundle plan" and "bundle deploy".
// Note, it is not run in "bundle migrate" so it must not modify the config
func PreDeployChecks(ctx context.Context, b *bundle.Bundle, isPlan bool, engine engine.EngineType) {
	bundle.ApplySeqContext(ctx, b,
		terraform.CheckDashboardsModifiedRemotely(isPlan, engine),
		resourcemutator.SecretScopeFixups(engine),
		deploy.StatePull(),
		mutator.ValidateGitDetails(),
		mutator.ValidateDirectOnlyResources(engine),
		mutator.ValidateLifecycleStarted(engine),
		mutator.ValidateCascadeOnDestroy(engine),
		statemgmt.CheckRunningResource(engine),
	)
}

// pipelineDeletionCascades reports whether deleting the pipeline referenced by a delete action
// also deletes its datasets (MVs, STs, Views). This is the server default (cascade) unless
// cascade_on_destroy is explicitly set to false. A lookup miss (unset field, or a non-pipeline
// resource) means the default applies, so it returns true.
// Reads from config like checkForPreventDestroy.
func pipelineDeletionCascades(b *bundle.Bundle, action deployplan.Action) bool {
	path, err := dyn.NewPathFromString(action.ResourceKey)
	if err != nil {
		return true
	}
	path = append(path, dyn.Key("cascade_on_destroy"))

	v, err := dyn.GetByPath(b.Config.Value(), path)
	if err != nil {
		return true
	}
	cascade, ok := v.AsBool()
	return !ok || cascade
}

// checkForPreventDestroy checks if the resource has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed.
// If it does, it returns an error.
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
