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
	Apply(context.Context, *Bundle) error
}

func Apply(ctx context.Context, b *Bundle, m Mutator) error {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))

	log.Debugf(ctx, "Apply")
	err := m.Apply(ctx, b)
	if err != nil {
		log.Errorf(ctx, "Error: %s", err)
		return err
	}

	return nil
}
