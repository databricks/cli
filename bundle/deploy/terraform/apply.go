package terraform

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/databricks/bricks/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type apply struct{}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	workingDir, err := Dir(b)
	if err != nil {
		return nil, err
	}

	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, err
	}

	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	err = tf.Apply(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform apply: %w", err)
	}

	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
