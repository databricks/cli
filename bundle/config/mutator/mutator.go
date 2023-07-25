package mutator

import (
	"github.com/databricks/cli/bundle"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		ProcessRootIncludes(),
		DefineDefaultEnvironment(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForEnvironment(env string) []bundle.Mutator {
	return append(DefaultMutators(), SelectEnvironment(env))
}
