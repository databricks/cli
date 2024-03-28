package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
)

// The load phase loads configuration from disk and performs
// lightweight preprocessing (anything that can be done without network I/O).
func Load() bundle.Mutator {
	return newPhase(
		"load",
		mutator.DefaultMutators(),
	)
}

func LoadDefaultTarget() bundle.Mutator {
	return newPhase(
		"load",
		append(mutator.DefaultMutators(), mutator.SelectDefaultTarget()),
	)
}

func LoadNamedTarget(target string) bundle.Mutator {
	return newPhase(
		"load",
		append(mutator.DefaultMutators(), mutator.SelectTarget(target)),
	)
}
