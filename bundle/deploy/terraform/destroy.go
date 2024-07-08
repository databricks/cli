package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

type destroy struct{}

func (w *destroy) Name() string {
	return "terraform.Destroy"
}

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// return early if plan is empty
	if b.Plan.IsEmpty() {
		cmdio.LogString(ctx, "No resources to destroy...")
		return nil
	}

	cmdio.LogString(ctx, "Destroying resources...")

	err := b.Plan.Apply()
	if err != nil {
		return diag.Errorf("terraform apply: %v", err)
	}
	return nil
}

// Destroy returns a [bundle.Mutator] that runs the conceptual equivalent of
// `terraform destroy ./plan` from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
