package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/aicode"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// LibLocationMap maps artifact names to library locations that need uploading.
// Computed by Build and consumed by Deploy to upload the right files.
type LibLocationMap map[string][]libraries.LocationToUpdate

// Build runs the build phase, which builds artifacts.
func Build(ctx context.Context, b *bundle.Bundle) LibLocationMap {
	log.Info(ctx, "Phase: build")

	bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreBuild),
		artifacts.Build(),
		scripts.Execute(config.ScriptPostBuild),

		// Package any AI Runtime task code_source_path that points at a local
		// directory into a content-addressed tarball overlaid on the sync root, and
		// rewrite the field to the synced workspace path. No requirements.yaml is
		// synthesized: the runtime installs pip deps from the job's serverless
		// environment (environments[].spec.dependencies) directly.
		aicode.PackageCodeSource(),

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
	)

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
