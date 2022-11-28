package bundle

import (
	"context"
)

// Mutator is the interface type that mutates a bundle's configuration or internal state.
// This makes every mutation or action observable and debuggable.
type Mutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply mutates the specified bundle object.
	// It may return a list of mutators to apply immediately after this mutator.
	// For example: when processing all configuration files in the tree; each file gets
	// its own mutator instance.
	Apply(context.Context, *Bundle) ([]Mutator, error)
}

func Apply(ctx context.Context, b *Bundle, ms []Mutator) error {
	if len(ms) == 0 {
		return nil
	}
	for _, m := range ms {
		ms_, err := m.Apply(ctx, b)
		if err != nil {
			return err
		}
		// Apply recursively.
		err = Apply(ctx, b, ms_)
		if err != nil {
			return err
		}
	}
	return nil
}
