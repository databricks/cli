package bundle

import (
	"context"
	"fmt"
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
	mainErr := Apply(ctx, b, d.mutators)
	errOnFinish := Apply(ctx, b, d.onFinishOrError)
	var err error = nil

	if mainErr != nil {
		err = mainErr
	}
	if errOnFinish != nil {
		if err == nil {
			err = errOnFinish
		} else {
			err = fmt.Errorf("%w\n%w", err, errOnFinish)
		}
	}
	return nil, err
}
