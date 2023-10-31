package mutator

import (
	"github.com/databricks/cli/bundle"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		// scripts.Execute(config.ScriptPreInit),
		ProcessRootIncludes(),
		DefineDefaultTarget(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForTarget(env string) []bundle.Mutator {
	return append(DefaultMutators(), SelectTarget(env))
}
