package bundle

import (
	"context"
	"sync"

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
		log.Debugf(ctx, "Error: %s", err)
	}

	return diags
}

func ApplySeq(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	diags := diag.Diagnostics{}
	for _, m := range mutators {
		diags = diags.Extend(Apply(ctx, b, m))
		if diags.HasError() {
			return diags
		}
	}
	return diags
}

// Run mutators in parallel. Unlike Apply and ApplySeq, this does not perform sync between
// dynamic and static configuration.
// Warning: none of the mutators involved must modify bundle directly or indirectly. In particular,
// the must not call bundle.Apply or bundle.ApplySeq because those include writes to config even if mutator does not.
// Deprecated: do not use for new use cases. Refactor your parallel task not to depend on bundle at all.
func ApplyParallelReadonly(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	var allDiags diag.Diagnostics
	resultsChan := make(chan diag.Diagnostics, len(mutators))
	var wg sync.WaitGroup

	contexts := make([]context.Context, len(mutators))

	for ind, m := range mutators {
		contexts[ind] = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))
		// log right away to have deterministic logger
		log.Debug(contexts[ind], "ApplyParallel")
	}

	for ind, m := range mutators {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// We're not using bundle.Apply here because we don't do copy between typed and dynamic values
			resultsChan <- m.Apply(contexts[ind], b)
		}()
	}

	wg.Wait()
	close(resultsChan)

	// Collect results into a single slice
	for diags := range resultsChan {
		allDiags = append(allDiags, diags...)
	}

	return allDiags
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
