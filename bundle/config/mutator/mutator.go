package mutator

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/bundle/scripts"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		scripts.Execute(config.ScriptPreInit),
		loader.ProcessRootIncludes(),
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
