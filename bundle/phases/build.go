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
	"github.com/databricks/cli/libs/logdiag"
)

// LibLocationMap maps artifact names to library locations that need uploading.
// Computed by Build and consumed by Deploy to upload the right files.
type LibLocationMap map[string][]libraries.LocationToUpdate

// resolveLibraries runs variable resolution, glob expansion, path rewriting,
// and wheel-task transformation to produce the local→remote upload map.
// extra mutators are applied after CheckForSameNameLibraries and before
// ReplaceWithRemotePath; Build passes libraries.SwitchToPatchedWheels() there.
func resolveLibraries(ctx context.Context, b *bundle.Bundle, extra ...bundle.Mutator) LibLocationMap {
	mutators := make([]bundle.Mutator, 0, 4+len(extra))
	mutators = append(mutators,
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
	)
	mutators = append(mutators, extra...)
	bundle.ApplySeqContext(ctx, b, mutators...)

	if logdiag.HasError(ctx) {
		return nil
	}

	libs, diags := libraries.ReplaceWithRemotePath(ctx, b)
	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
	bundle.ApplyContext(ctx, b, trampoline.TransformWheelTask())
	return libs
}

// Build runs the build phase, which builds artifacts.
func Build(ctx context.Context, b *bundle.Bundle) LibLocationMap {
	log.Info(ctx, "Phase: build")

	bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreBuild),
		artifacts.Build(),
		scripts.Execute(config.ScriptPostBuild),
	)

	if logdiag.HasError(ctx) {
		return nil
	}

	// SwitchToPatchedWheels must be passed to resolveLibraries so it runs after
	// ExpandGlobReferences (which expands *.whl patterns in job library paths)
	// and after the build phase (which populates Artifact.Source and Artifact.Patched).
	return resolveLibraries(ctx, b, libraries.SwitchToPatchedWheels())
}

// FindLibraries discovers which local library files need uploading by reading
// the bundle config (glob expansion and path rewriting) without running any
// build commands. Used when applying a saved plan: artifacts were already built
// at plan time and the plan's new_state carries the correct remote paths.
func FindLibraries(ctx context.Context, b *bundle.Bundle) LibLocationMap {
	log.Info(ctx, "Phase: find libraries")
	return resolveLibraries(ctx, b)
}
