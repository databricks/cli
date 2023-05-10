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
	if mainErr == nil && errOnFinish == nil {
		return nil, nil
	}

	err := fmt.Errorf("Error")
	if mainErr != nil {
		err = fmt.Errorf("%w\n%v", err, mainErr)
	}
	if errOnFinish != nil {
		err = fmt.Errorf("%w\n%v", err, errOnFinish)
	}
	return nil, err
}
