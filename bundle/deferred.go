package bundle

import (
	"context"

	"github.com/databricks/cli/libs/errs"
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

func (d *DeferredMutator) Apply(ctx context.Context, b *Bundle) error {
	mainErr := Apply(ctx, b, d.mutator)
	errOnFinish := Apply(ctx, b, d.finally)
	if mainErr != nil || errOnFinish != nil {
		return errs.FromMany(mainErr, errOnFinish)
	}

	return nil
}
