package bundle

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// Mutator is the interface type that mutates a bundle's configuration or internal state.
// This makes every mutation or action observable and debuggable.
type Mutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply mutates the specified bundle object.
	Apply(context.Context, *Bundle) diag.Diagnostics
}

func ApplyContext(ctx context.Context, b *Bundle, m Mutator) {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))

	log.Debugf(ctx, "Apply")

	err := b.Config.MarkMutatorEntry(ctx)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("entry error: %w", err))
		return
	}

	defer func() {
		err := b.Config.MarkMutatorExit(ctx)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("exit error: %w", err))
		}
	}()

	diags := m.Apply(ctx, b)

	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
}

func ApplySeqContext(ctx context.Context, b *Bundle, mutators ...Mutator) {
	for _, m := range mutators {
		ApplyContext(ctx, b, m)
		if logdiag.HasError(ctx) {
			break
		}
	}
}

type funcMutator struct {
	fn func(context.Context, *Bundle)
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	m.fn(ctx, b)
	return nil
}

// ApplyFuncContext applies an inline-specified function mutator.
func ApplyFuncContext(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle)) {
	ApplyContext(ctx, b, funcMutator{fn})
}

// Test helpers. TODO: move to separate package.

func Apply(ctx context.Context, b *Bundle, m Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	logdiag.SetCollect(ctx, true)
	ApplyContext(ctx, b, m)
	return logdiag.FlushCollected(ctx)
}

// Test helper to get diagnostics in this call
func ApplySeq(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	logdiag.SetCollect(ctx, true)
	ApplySeqContext(ctx, b, mutators...)
	return logdiag.FlushCollected(ctx)
}
