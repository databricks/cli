package terraform

import (
	"context"

	"github.com/databricks/bricks/bundle"
)

type load struct{}

func (l *load) Name() string {
	return "terraform.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	state, err := b.Terraform.Show(ctx)
	if err != nil {
		return nil, err
	}

	// Merge state into configuration.
	err = TerraformToBundle(state, &b.Config)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func Load() bundle.Mutator {
	return &load{}
}
