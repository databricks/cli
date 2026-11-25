package stages

import "github.com/databricks/cli/bundle"

// MutatorNamesByStage returns the names of the registered mutators per stage, in the
// order they run. It lets the order test in package stages_test, which imports the
// packages that register mutators, read the registry.
func MutatorNamesByStage() map[Stage][]string {
	return mutatorNames(mutatorsByStage())
}

func mutatorNames(byStage map[Stage][]func() bundle.Mutator) map[Stage][]string {
	names := map[Stage][]string{}
	for stage, newMutators := range byStage {
		for _, newMutator := range newMutators {
			names[stage] = append(names[stage], newMutator().Name())
		}
	}
	return names
}
