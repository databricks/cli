package loader

import (
	"context"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/mutator"
)

func ConfigureForEnvironment(ctx context.Context, env string) (context.Context, error) {
	b, err := bundle.LoadFromRoot()
	if err != nil {
		return nil, err
	}

	err = bundle.Apply(ctx, b, mutator.DefaultMutatorsForEnvironment(env))
	if err != nil {
		return nil, err
	}

	return bundle.Context(ctx, b), nil
}
