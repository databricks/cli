package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type validate struct{}

type location struct {
	path string
	rb   bundle.ReadOnlyBundle
}

func (l location) Location() dyn.Location {
	return l.rb.Config().GetLocation(l.path)
}

func (l location) Locations() []dyn.Location {
	return l.rb.Config().GetLocations(l.path)
}

func (l location) Path() dyn.Path {
	return dyn.MustPathFromString(l.path)
}

// Apply implements bundle.Mutator.
func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), bundle.Parallel(
		FastValidateReadonly(),

		// Slow mutators that require network or file i/o. These are only
		// run in the `bundle validate` command.
		FilesToSync(),
		ValidateFolderPermissions(),
		ValidateSyncPatterns(),
	))
}

// Name implements bundle.Mutator.
func (v *validate) Name() string {
	return "validate"
}

func Validate() bundle.Mutator {
	return &validate{}
}
