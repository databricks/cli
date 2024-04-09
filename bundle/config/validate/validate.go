package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type validate struct {
}

type location struct {
	path string
	b    *bundle.Bundle
}

func (l location) Location() dyn.Location {
	return l.b.Config.GetLocation(l.path)
}

func (l location) Path() dyn.Path {
	return dyn.MustPathFromString(l.path)
}

// Apply implements bundle.Mutator.
func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.Apply(ctx, b, bundle.Parallel(
		JobClusterKeyDefined(),
		FilesToSync(),
	))
}

// Name implements bundle.Mutator.
func (v *validate) Name() string {
	return "validate"
}

func Validate() bundle.Mutator {
	return &validate{}
}
