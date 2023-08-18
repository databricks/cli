package mutator

import (
	"github.com/databricks/cli/bundle"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		ProcessRootIncludes(),
		DefineDefaultTarget(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForTarget(env string) []bundle.Mutator {
	return append(DefaultMutators(), SelectTarget(env))
}
