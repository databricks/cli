package mutator

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/scripts"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		scripts.Execute(config.ScriptPreInit),
		ProcessRootIncludes(),
		InitializeVariables(),
		DefineDefaultTarget(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForTarget(env string) []bundle.Mutator {
	return append(DefaultMutators(), SelectTarget(env))
}
