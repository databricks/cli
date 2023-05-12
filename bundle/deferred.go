package bundle

import (
	"context"

	"github.com/databricks/bricks/libs/errors"
)



type DeferredMutator struct {
	mutators        []Mutator
	finally []Mutator
}

func (d *DeferredMutator) Name() string {
	return "deferred"
}

func Defer(mutators []Mutator, finally []Mutator) []Mutator {
	return []Mutator{
		&DeferredMutator{
			mutators:        mutators,
			finally: finally,
		},
	}
}

func (d *DeferredMutator) Apply(ctx context.Context, b *Bundle) ([]Mutator, error) {
	mainErr := Apply(ctx, b, d.mutators)
	errOnFinish := Apply(ctx, b, d.finally)
	if mainErr != nil || errOnFinish != nil {
		return nil, errors.FromMany(mainErr, errOnFinish)
	}

	return nil, nil
}
