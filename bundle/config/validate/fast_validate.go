package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/stages"
	"github.com/databricks/cli/libs/diag"
)

// FastValidate runs a subset of fast validation checks. This is a subset of the full
// suite of validation mutators that satisfy ANY ONE of the following criteria:
//
// 1. No file i/o or network requests are made in the mutator.
// 2. The validation is blocking for bundle deployments.
//
// The full suite of validation mutators is available in the [Validate] mutator.
type fastValidate struct{ bundle.RO }

func FastValidate() bundle.ReadOnlyMutator {
	return &fastValidate{}
}

func (f *fastValidate) Name() string {
	return "fast_validate(readonly)"
}

func (f *fastValidate) Apply(ctx context.Context, rb *bundle.Bundle) diag.Diagnostics {
	bundle.ApplyParallel(ctx, rb, append(
		// Fast mutators with only in-memory checks
		readOnlyStageMutators(stages.StageFastValidation),

		// Blocking mutators. Deployments will fail if these checks fail.
		ValidateArtifactPath(),
	)...)

	return nil
}

// readOnlyStageMutators returns the mutators registered for stage, for a caller that
// runs them in parallel. It panics on a mutator that is not read-only, because running
// it in parallel would race on the bundle.
func readOnlyStageMutators(stage stages.Stage) []bundle.ReadOnlyMutator {
	var mutators []bundle.ReadOnlyMutator
	for _, m := range stages.Mutators(stage) {
		readOnly, ok := m.(bundle.ReadOnlyMutator)
		if !ok {
			panic(fmt.Sprintf("mutator %q registered for stage %q must be read-only, because the stage runs in parallel", m.Name(), stage))
		}
		mutators = append(mutators, readOnly)
	}
	return mutators
}
