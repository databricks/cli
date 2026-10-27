package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/libs/logdiag"
)

func DefaultMutators(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplySeqContext(ctx, b,
		loader.EntryPoint(),

		// Execute preinit script before processing includes.
		// It needs to be done before processing configuration files to allow
		// the script to modify the configuration or add own configuration files.
		scripts.Execute(config.ScriptPreInit),
		loader.ProcessRootIncludes(),

		// Verify that the CLI version is within the specified range.
		VerifyCliVersion(),

		EnvironmentsToTargets(),
		ComputeIdToClusterId(),
		InitializeVariables(),
		DefineDefaultTarget(),

		// Note: These mutators must run before the target overrides are merged.
		// See the mutators for more details.
		validate.NoVariableReferenceInResourceKey(),
		validate.UniqueResourceKeys(),
	)
}

// FinishLoading converts the configuration of b, loaded and merged by
// [DefaultMutators] (and a target selection), into the typed configuration. It is not
// a mutator: it runs once, at the end of the load phase, also if loading failed so
// that the configuration can still be inspected.
func FinishLoading(ctx context.Context, b *bundle.Bundle) {
	diags, err := b.Config.FinishLoading()
	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}
