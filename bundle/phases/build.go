package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/log"
)

type LibLocationMap map[string][]libraries.LocationToUpdate

// The build phase builds artifacts.
func Build(ctx context.Context, b *bundle.Bundle) (LibLocationMap, error) {
	log.Info(ctx, "Phase: build")

	if err := bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreBuild),
		artifacts.Build(),
		scripts.Execute(config.ScriptPostBuild),
		mutator.ResolveVariableReferencesWithoutResources(
			"artifacts",
		),
		mutator.ResolveVariableReferencesOnlyResources(
			"artifacts",
		),

		// libraries.CheckForSameNameLibraries() needs to be run after we expand glob references so we
		// know what are the actual library paths.
		// libraries.ExpandGlobReferences() has to be run after the libraries are built and thus this
		// mutator is part of the deploy step rather than validate.
		libraries.ExpandGlobReferences(),
		libraries.CheckForSameNameLibraries(),
		// SwitchToPatchedWheels must be run after ExpandGlobReferences and after build phase because it Artifact.Source and Artifact.Patched populated
		libraries.SwitchToPatchedWheels(),
	); err != nil {
		return nil, err
	}

	libs, err := libraries.ReplaceWithRemotePath(ctx, b)
	if err != nil {
		return nil, err
	}

	if err := bundle.ApplyContext(ctx, b,
		// TransformWheelTask must be run after ReplaceWithRemotePath so we can use correct remote path in the
		// transformed notebook
		trampoline.TransformWheelTask(),
	); err != nil {
		return nil, err
	}

	return libs, nil
}
