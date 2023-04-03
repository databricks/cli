package terraform

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type destroy struct{}

func (w *destroy) Name() string {
	return "terraform.Destroy"
}

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if !b.ConfirmDestroy {
		return nil, nil
	}

	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	err = tf.Destroy(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform destroy: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Successfully destroyed resources!")
	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
