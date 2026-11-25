package stages

// MutatorNamesByStage returns the names of the registered mutators per stage, in the
// order they run. It lets the order test in package stages_test, which imports the
// packages that register mutators, read the registry.
func MutatorNamesByStage() map[Stage][]string {
	actual := map[Stage][]string{}
	for stage, mutators := range stageMutatorsByStage() {
		for _, m := range mutators {
			actual[stage] = append(actual[stage], m.newMutator().Name())
		}
	}
	return actual
}
