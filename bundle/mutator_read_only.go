package bundle

import (
	"context"
	"sync"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

type ReadOnlyMutator interface {
	Mutator

	// This is just tag, to differentiate this interface from bundle.Mutator
	// This prevents non-readonly mutators being passed to ApplyParallel().
	IsRO()
}

// Helper to mark the mutator as "read-only"
type RO struct{}

func (*RO) IsRO() {}

// Run mutators in parallel. Unlike Apply and ApplySeq, this does not perform sync between
// dynamic and static configuration.
// Warning: none of the mutators involved must modify bundle directly or indirectly. In particular,
// they must not call bundle.ApplyContext or bundle.ApplyContextSeq because those include writes to config even if mutator does not.
// Deprecated: do not use for new use cases. Refactor your parallel task not to depend on bundle at all.
func ApplyParallel(ctx context.Context, b *Bundle, mutators ...ReadOnlyMutator) {
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
			// We're not using bundle.ApplyContext here because we don't do copy between typed and dynamic values
			diags := m.Apply(contexts[ind], b)
			for _, d := range diags {
				logdiag.LogDiag(ctx, d)
			}
		}()
	}

	wg.Wait()
}
