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
		statemgmt.CheckRunningResource(engine),
	)
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

		// If there is no prevent_destroy, skip
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
