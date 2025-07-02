package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/scripts"
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

		// Note: This mutator must run before the target overrides are merged.
		// See the mutator for more details.
		validate.UniqueResourceKeys(),
	)
}
