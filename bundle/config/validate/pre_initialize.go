package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type preInitialize struct{}

// Apply implements bundle.Mutator.
func (v *preInitialize) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), bundle.Parallel(
		UniqueResourceKeys(),
	))
}

// Name implements bundle.Mutator.
func (v *preInitialize) Name() string {
	return "validate:pre_initialize"
}

// Validations to perform before initialization of the bundle. These validations
// are thus applied for most bundle commands.
func PreInitialize() bundle.Mutator {
	return &preInitialize{}
}
