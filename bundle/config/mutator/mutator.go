package mutator

import "github.com/databricks/bricks/bundle/config"

// Mutator is the interface types that mutate the bundle configuration.
// This makes every mutation observable and debuggable.
type Mutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply mutates the specified configuration object.
	// It optionally returns a list of mutators to invoke immediately after this mutator.
	// This is used when processing all configuration files in the tree; each file gets
	// its own mutator instance.
	Apply(*config.Root) ([]Mutator, error)
}

func DefaultMutators() []Mutator {
	return []Mutator{
		DefineDefaultInclude(),
		ProcessRootIncludes(),
		DefineDefaultEnvironment(),
	}
}

func DefaultMutatorsForEnvironment(env string) []Mutator {
	return append(DefaultMutators(), SelectEnvironment(env))
}

func Apply(root *config.Root, ms []Mutator) error {
	if len(ms) == 0 {
		return nil
	}
	for _, m := range ms {
		ms_, err := m.Apply(root)
		if err != nil {
			return err
		}
		// Apply recursively.
		err = Apply(root, ms_)
		if err != nil {
			return err
		}
	}
	return nil
}
