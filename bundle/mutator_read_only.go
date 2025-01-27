package bundle

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// ReadOnlyMutator is the interface type that allows access to bundle configuration but does not allow any mutations.
type ReadOnlyMutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply access the specified read-only bundle object.
	Apply(context.Context, ReadOnlyBundle) diag.Diagnostics
}

func ApplyReadOnly(ctx context.Context, rb ReadOnlyBundle, m ReadOnlyMutator) diag.Diagnostics {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator (read-only)", m.Name()))

	log.Debugf(ctx, "ApplyReadOnly")
	diags := m.Apply(ctx, rb)
	if err := diags.Error(); err != nil {
		log.Debugf(ctx, "Error: %s", err)
	}

	return diags
}
