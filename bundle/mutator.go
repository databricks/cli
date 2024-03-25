package bundle

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// Mutator is the interface type that mutates a bundle's configuration or internal state.
// This makes every mutation or action observable and debuggable.
type Mutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply mutates the specified bundle object.
	Apply(context.Context, *Bundle) diag.Diagnostics
}

func Apply(ctx context.Context, b *Bundle, m Mutator) diag.Diagnostics {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))

	log.Debugf(ctx, "Apply")

	err := b.Config.MarkMutatorEntry(ctx)
	if err != nil {
		log.Errorf(ctx, "entry error: %s", err)
		return diag.Errorf("entry error: %s", err)
	}

	defer func() {
		err := b.Config.MarkMutatorExit(ctx)
		if err != nil {
			log.Errorf(ctx, "exit error: %s", err)
		}
	}()

	diags := m.Apply(ctx, b)

	// Log error in diagnostics if any.
	// Note: errors should be logged when constructing them
	// such that they are not logged multiple times.
	// If this is done, we can omit this block.
	if err := diags.Error(); err != nil {
		log.Errorf(ctx, "Error: %s", err)
	}

	return diags
}

type funcMutator struct {
	fn func(context.Context, *Bundle) diag.Diagnostics
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	return m.fn(ctx, b)
}

// ApplyFunc applies an inline-specified function mutator.
func ApplyFunc(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle) diag.Diagnostics) diag.Diagnostics {
	return Apply(ctx, b, funcMutator{fn})
}
