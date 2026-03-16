package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
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
	bundle.ApplyParallel(ctx, rb,
		// Fast mutators with only in-memory checks
		JobClusterKeyDefined(),
		JobTaskClusterSpec(),

		// Blocking mutators. Deployments will fail if these checks fail.
		ValidateArtifactPath(),
	)

	return nil
}
