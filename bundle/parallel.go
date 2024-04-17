package bundle

import (
	"context"
	"sync"

	"github.com/databricks/cli/libs/diag"
)

type parallel struct {
	mutators []ReadOnlyMutator
}

func (m *parallel) Name() string {
	return "parallel"
}

func (m *parallel) Apply(ctx context.Context, rb ReadOnlyBundle) diag.Diagnostics {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var diags diag.Diagnostics

	wg.Add(len(m.mutators))
	for _, mutator := range m.mutators {
		go func(mutator ReadOnlyMutator) {
			defer wg.Done()
			d := ApplyReadOnly(ctx, rb, mutator)

			mu.Lock()
			diags = diags.Extend(d)
			mu.Unlock()
		}(mutator)
	}
	wg.Wait()
	return diags
}

// Parallel runs the given mutators in parallel.
func Parallel(mutators ...ReadOnlyMutator) ReadOnlyMutator {
	return &parallel{
		mutators: mutators,
	}
}
