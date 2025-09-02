package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
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

// deployPrepare is common set of mutators between "bundle plan" and "bundle deploy".
// This function does not make any mutations in the workspace remotely, only in-memory bundle config mutations
func deployPrepare(ctx context.Context, b *bundle.Bundle) map[string][]libraries.LocationToUpdate {
	bundle.ApplySeqContext(ctx, b,
		statemgmt.StatePull(),
		terraform.CheckDashboardsModifiedRemotely(),
		deploy.StatePull(),
		mutator.ValidateGitDetails(),
		terraform.CheckRunningResource(),

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
func checkForPreventDestroy(b *bundle.Bundle, actions []deployplan.Action, isDestroy bool) error {
	root := b.Config.Value()
	for _, action := range actions {
		if action.ActionType == deployplan.ActionTypeRecreate || (isDestroy && action.ActionType == deployplan.ActionTypeDelete) {
			path := dyn.NewPath(dyn.Key("resources"), dyn.Key(action.Group), dyn.Key(action.Key), dyn.Key("lifecycle"))
			lifecycleV, err := dyn.GetByPath(root, path)
			// If there is no lifecycle, skip
			if err != nil {
				return nil
			}

			if lifecycleV.Kind() == dyn.KindMap {
				preventDestroyV := lifecycleV.Get("prevent_destroy")
				preventDestroy, ok := preventDestroyV.AsBool()
				if ok && preventDestroy {
					return fmt.Errorf("resource %s has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed. To avoid this error, disable lifecycle.prevent_destroy", action.Key)
				}
			}
		}
	}
	return nil
}
