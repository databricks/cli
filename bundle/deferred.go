package bundle

import (
	"context"
	"errors"
)

type DeferredMutator struct {
	mutators        []Mutator
	onFinishOrError []Mutator
}

func (d *DeferredMutator) Name() string {
	return "deferred"
}

func Defer(mutators []Mutator, onFinishOrError []Mutator) []Mutator {
	return []Mutator{
		&DeferredMutator{
			mutators:        mutators,
			onFinishOrError: onFinishOrError,
		},
	}
}

func (d *DeferredMutator) Apply(ctx context.Context, b *Bundle) ([]Mutator, error) {
	err := Apply(ctx, b, d.mutators)
	errOnFinish := Apply(ctx, b, d.onFinishOrError)

	if err != nil || errOnFinish != nil {
		return nil, errors.Join(err, errOnFinish)
	}
	return nil, nil
}
