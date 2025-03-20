package bundle

import (
	"context"
	"sync"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type ReadOnlyMutator interface {
	Mutator

	// This is just tag, to differentiate this interface from bundle.Mutator
	// This prevents non-readonly mutators being passed to ApplyParallel().
	// IsRO()
}

// Helper to mark the mutator as "read-only"
type RO struct{}

func (*RO) IsRO() {}

// Run mutators in parallel. Unlike Apply and ApplySeq, this does not perform sync between
// dynamic and static configuration.
// Warning: none of the mutators involved must modify bundle directly or indirectly. In particular,
// they must not call bundle.Apply or bundle.ApplySeq because those include writes to config even if mutator does not.
// Deprecated: do not use for new use cases. Refactor your parallel task not to depend on bundle at all.
func ApplyParallel(ctx context.Context, b *Bundle, mutators ...ReadOnlyMutator) diag.Diagnostics {
	var allDiags diag.Diagnostics
	resultsChan := make(chan diag.Diagnostics, len(mutators))
	var wg sync.WaitGroup

	contexts := make([]context.Context, len(mutators))

	for ind, m := range mutators {
		contexts[ind] = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))
		// log right away to have deterministic order of log messages
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
