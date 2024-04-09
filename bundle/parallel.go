package bundle

import (
	"context"
	"sync"

	"github.com/databricks/cli/libs/diag"
)

type parallel struct {
	mutators []Mutator
}

func (m *parallel) Name() string {
	return "parallel"
}

func (m *parallel) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	var wg sync.WaitGroup
	diags := diag.Diagnostics{}
	wg.Add(len(m.mutators))
	for _, mutator := range m.mutators {
		go func(mutator Mutator) {
			defer wg.Done()
			diags = diags.Extend(mutator.Apply(ctx, b))
		}(mutator)
	}
	wg.Wait()
	return diags
}

// Parallel runs the given mutators in parallel.
func Parallel(mutators ...Mutator) Mutator {
	return &parallel{
		mutators: mutators,
	}
}
