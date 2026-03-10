package phases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
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
		statemgmt.CheckRunningResource(engine),
	)
}

// checkForPreventDestroy checks if the resource has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed.
// If it does, it returns an error.
func checkForPreventDestroy(b *bundle.Bundle, actions []deployplan.Action) error {
	var errs []error
	for _, action := range actions {
		if action.ActionType != deployplan.Recreate && action.ActionType != deployplan.Delete {
			continue
		}

		// ResourceKey format: "resources.{type}.{key}"
		parts := strings.SplitN(action.ResourceKey, ".", 3)
		if len(parts) != 3 || parts[0] != "resources" {
			continue
		}
		resourceType, resourceKey := parts[1], parts[2]

		for _, group := range b.Config.Resources.AllResources() {
			if group.Description.PluralName != resourceType {
				continue
			}
			if r, ok := group.Resources[resourceKey]; ok && r.GetLifecycle().HasPreventDestroy() {
				errs = append(errs, fmt.Errorf("%s has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed. To avoid this error, disable lifecycle.prevent_destroy for %s", action.ResourceKey, action.ResourceKey))
			}
			break
		}
	}

	return errors.Join(errs...)
}
