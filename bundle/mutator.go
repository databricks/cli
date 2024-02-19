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

	err := b.Config.MarkMutatorEntry(ctx)
	if err != nil {
		log.Errorf(ctx, "entry error: %s", err)
		return err
	}

	defer func() {
		err := b.Config.MarkMutatorExit(ctx)
		if err != nil {
			log.Errorf(ctx, "exit error: %s", err)
		}
	}()

	err = m.Apply(ctx, b)
	if err != nil {
		log.Errorf(ctx, "Error: %s", err)
		return err
	}

	return nil
}

type funcMutator struct {
	fn func(context.Context, *Bundle) error
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) error {
	return m.fn(ctx, b)
}

// ApplyFunc applies an inline-specified function mutator.
func ApplyFunc(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle) error) error {
	return Apply(ctx, b, funcMutator{fn})
}
