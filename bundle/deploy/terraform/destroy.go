package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type destroy struct{}

func (w *destroy) Name() string {
	return "terraform.Destroy"
}

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// return early if plan is empty
	if b.Plan.IsEmpty {
		log.Debugf(ctx, "No resources to destroy in plan. Skipping destroy.")
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	if b.Plan.Path == "" {
		return diag.Errorf("no plan found")
	}

	// Apply terraform according to the computed destroy plan
	err := tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return diag.Errorf("terraform destroy: %v", err)
	}
	return nil
}

// Destroy returns a [bundle.Mutator] that runs the conceptual equivalent of
// `terraform destroy ./plan` from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
