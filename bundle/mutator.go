package bundle

import (
	"context"

	"github.com/databricks/cli/libs/log"
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

// applyMutator calls apply on the specified mutator given a bundle.
// Any mutators this call returns are applied recursively.
func applyMutator(ctx context.Context, b *Bundle, m Mutator) error {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))

	log.Debugf(ctx, "Apply")
	ms, err := m.Apply(ctx, b)
	if err != nil {
		log.Errorf(ctx, "Error: %s", err)
		return err
	}

	// Apply recursively.
	err = Apply(ctx, b, ms)
	if err != nil {
		return err
	}

	return nil
}

func Apply(ctx context.Context, b *Bundle, ms []Mutator) error {
	if len(ms) == 0 {
		return nil
	}
	for _, m := range ms {
		err := applyMutator(ctx, b, m)
		if err != nil {
			return err
		}
	}
	return nil
}
