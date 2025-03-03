package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validate struct{}

func Validate(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.ApplyParallel(ctx, b,
		FastValidate(),

		// Slow mutators that require network or file i/o. These are only
		// run in the `bundle validate` command.
		FilesToSync(),
		ValidateFolderPermissions(),
		ValidateSyncPatterns(),
	)
}

// Name implements bundle.Mutator.
func (v *validate) Name() string {
	return "validate"
}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return Validate(ctx, b)
}

func NewValidateMutator() bundle.Mutator {
	return &validate{}
}
