package ucm

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// Mutator is a function-object that transforms a Ucm's configuration tree or
// runtime state. Mirrors bundle.Mutator so converting code between the two
// stays mechanical.
type Mutator interface {
	// Name returns the mutator's display name.
	Name() string

	// Apply runs the mutation against u.
	Apply(context.Context, *Ucm) diag.Diagnostics
}

// ApplyContext runs a single mutator, keeping the dynamic/typed trees in sync
// and forwarding diagnostics to logdiag.
func ApplyContext(ctx context.Context, u *Ucm, m Mutator) {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))
	log.Debugf(ctx, "Apply")

	if err := u.Config.MarkMutatorEntry(ctx); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("entry error: %w", err))
		return
	}
	defer func() {
		if err := u.Config.MarkMutatorExit(ctx); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("exit error: %w", err))
		}
	}()

	for _, d := range m.Apply(ctx, u) {
		logdiag.LogDiag(ctx, d)
	}
}

// ApplySeqContext runs mutators in order, stopping at the first one that logs
// an error.
func ApplySeqContext(ctx context.Context, u *Ucm, mutators ...Mutator) {
	for _, m := range mutators {
		ApplyContext(ctx, u, m)
		if logdiag.HasError(ctx) {
			break
		}
	}
}

type funcMutator struct {
	fn func(context.Context, *Ucm)
}

func (m funcMutator) Name() string { return "<func>" }

func (m funcMutator) Apply(ctx context.Context, u *Ucm) diag.Diagnostics {
	m.fn(ctx, u)
	return nil
}

// ApplyFuncContext applies an inline-specified function mutator.
func ApplyFuncContext(ctx context.Context, u *Ucm, fn func(context.Context, *Ucm)) {
	ApplyContext(ctx, u, funcMutator{fn})
}

// Apply runs a single mutator in a test-friendly way: it ensures logdiag is
// set up, collects diagnostics into a slice, and returns them.
func Apply(ctx context.Context, u *Ucm, m Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplyContext(ctx, u, m)
	return logdiag.FlushCollected(ctx)
}

// ApplySeq is the test-helper equivalent of ApplySeqContext.
func ApplySeq(ctx context.Context, u *Ucm, mutators ...Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplySeqContext(ctx, u, mutators...)
	return logdiag.FlushCollected(ctx)
}
