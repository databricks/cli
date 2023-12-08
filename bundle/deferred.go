package bundle

import (
	"context"

	"github.com/databricks/cli/libs/errs"
)

type DeferredMutator struct {
	mutator Mutator
	finally Mutator
}
type contextKey string

const mainErrKey contextKey = "mainErr"

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
	if mainErr != nil {
		ctx = context.WithValue(ctx, mainErrKey, mainErr)
	}
	errOnFinish := Apply(ctx, b, d.finally)
	if mainErr != nil || errOnFinish != nil {
		return errs.FromMany(mainErr, errOnFinish)
	}

	return nil
}

func ErrFromContext(ctx context.Context) error {
	if err, ok := ctx.Value(mainErrKey).(error); ok {
		return err
	}
	return nil
}
