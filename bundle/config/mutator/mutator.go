package mutator

import (
	"github.com/databricks/bricks/bundle"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		DefineDefaultInclude(),
		ProcessRootIncludes(),
		DefineDefaultEnvironment(),
		LoadGitConfig(),
	}
}

func DefaultMutatorsForEnvironment(env string) []bundle.Mutator {
	return append(DefaultMutators(), SelectEnvironment(env))
}
