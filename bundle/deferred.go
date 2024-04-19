package bundle

import (
	"context"

	"github.com/databricks/cli/libs/diag"
)

type DeferredMutator struct {
	mutator Mutator
	finally Mutator
}

func (d *DeferredMutator) Name() string {
	return "deferred"
}

func Defer(mutator Mutator, finally Mutator) Mutator {
	return &DeferredMutator{
		mutator: mutator,
		finally: finally,
	}
}

func (d *DeferredMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	diags = diags.Extend(Apply(ctx, b, d.mutator))
	diags = diags.Extend(Apply(ctx, b, d.finally))
	return diags
}
