package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

// DeployPrepare is common set of mutators between "bundle plan" and "bundle deploy".
// This function does not make any mutations in the workspace remotely, only in-memory bundle config mutations
func DeployPrepare(ctx context.Context, b *bundle.Bundle, isPlan bool, engine engine.EngineType) map[string][]libraries.LocationToUpdate {
	bundle.ApplySeqContext(ctx, b,
		terraform.CheckDashboardsModifiedRemotely(isPlan, engine),
		deploy.StatePull(),
		mutator.ValidateGitDetails(),
		statemgmt.CheckRunningResource(engine),

		// libraries.CheckForSameNameLibraries() needs to be run after we expand glob references so we
		// know what are the actual library paths.
		// libraries.ExpandGlobReferences() has to be run after the libraries are built and thus this
		// mutator is part of the deploy step rather than validate.
		libraries.ExpandGlobReferences(),
		libraries.CheckForSameNameLibraries(),
		// SwitchToPatchedWheels must be run after ExpandGlobReferences and after build phase because it Artifact.Source and Artifact.Patched populated
		libraries.SwitchToPatchedWheels(),
	)

	libs, diags := libraries.ReplaceWithRemotePath(ctx, b)
	for _, diag := range diags {
		logdiag.LogDiag(ctx, diag)
	}

	bundle.ApplySeqContext(ctx, b,
		// TransformWheelTask must be run after ReplaceWithRemotePath so we can use correct remote path in the
		// transformed notebook
		trampoline.TransformWheelTask(),
	)

	return libs
}

// checkForPreventDestroy checks if the resource has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed.
// If it does, it returns an error.
func checkForPreventDestroy(b *bundle.Bundle, actions []deployplan.Action) error {
	root := b.Config.Value()
	var errs []error
	for _, action := range actions {
		if action.ActionType != deployplan.ActionTypeRecreate && action.ActionType != deployplan.ActionTypeDelete {
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
