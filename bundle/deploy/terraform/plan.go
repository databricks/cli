package terraform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type plan struct {
	destroy bool
}

func (p *plan) Name() string {
	return "terraform.Plan"
}

func (p *plan) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	// Persist computed plan
	tfDir, err := Dir(b)
	if err != nil {
		return nil, err
	}
	planPath := filepath.Join(tfDir, "plan")
	notEmpty, err := tf.Plan(ctx, tfexec.Destroy(p.destroy), tfexec.Out(planPath))
	if err != nil {
		return nil, err
	}

	// Set plan in main bundle struct for downstream mutators
	// TODO: allow bypass using cmd line flag
	b.Plan = &terraform.Plan{
		Path:         planPath,
		ConfirmApply: false,
		IsEmpty:      !notEmpty,
	}
	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Plan(destroy bool) bundle.Mutator {
	return &plan{
		destroy: destroy,
	}
}
