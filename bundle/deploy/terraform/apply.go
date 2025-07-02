package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type apply struct{}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// return early if plan is empty
	if b.Plan.TerraformIsEmpty {
		log.Debugf(ctx, "No changes in plan. Skipping terraform apply.")
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	if b.Plan.TerraformPlanPath == "" {
		return diag.Errorf("no plan found")
	}

	// Apply terraform according to the computed plan
	err := tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.TerraformPlanPath))
	if err != nil {
		diags := permissions.TryExtendTerraformPermissionError(ctx, b, err)
		if diags != nil {
			return diags
		}
		return diag.Errorf("terraform apply: %v", err)
	}

	log.Infof(ctx, "terraform apply completed")
	return nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply ./plan`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
