package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type apply struct{}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	cmdio.LogMutatorEvent(ctx, w.Name(), cmdio.MutatorRunning, "Starting resource deployment")

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	err = tf.Apply(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform apply: %w", err)
	}

	cmdio.LogMutatorEvent(ctx, w.Name(), cmdio.MutatorCompleted, "Resource deployment completed!\n")
	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
