package mutator

import (
	"github.com/databricks/cli/bundle"
)

var defaultMutators []bundle.Mutator = []bundle.Mutator{
	ProcessRootIncludes(),
	DefineDefaultTarget(),
	LoadGitDetails(),
}

func DefaultMutators() []bundle.Mutator {
	return append(defaultMutators, SetRunAs())
}

func DefaultMutatorsForTarget(env string) []bundle.Mutator {
	return append(defaultMutators, SelectTarget(env), SetRunAs())
}
