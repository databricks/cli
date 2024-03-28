package mutator

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/bundle/scripts"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		loader.EntryPoint(),
		loader.ProcessRootIncludes(),

		// Verify that the CLI version is within the specified range.
		VerifyCliVersion(),

		// Execute preinit script after loading all configuration files.
		scripts.Execute(config.ScriptPreInit),
		EnvironmentsToTargets(),
		InitializeVariables(),
		DefineDefaultTarget(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForTarget(target string) []bundle.Mutator {
	return append(
		DefaultMutators(),
		SelectTarget(target),
	)
}
