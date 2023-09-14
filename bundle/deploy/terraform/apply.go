package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type apply struct{}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) error {
	// Error if terraform is not initialized
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	// Error if plan is missing
	if b.Plan.Path == "" {
		return fmt.Errorf("no path specified for computed plan")
	}

	// Print the plan if it's not empty
	if !b.Plan.IsEmpty {
		// parse the computed terraform plan
		plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
		if err != nil {
			return err
		}

		// print a high level summary of the terraform plan
		err = printPlanSummary(ctx, plan)
		if err != nil {
			return err
		}
	}

	if !b.Plan.IsEmpty {
		// We do not block for confirmation checks at deploy
		cmdio.LogString(ctx, "Deploying resources")
	}

	// Apply terraform plan. Note: we apply even if a plan is empty to generate
	// an empty state file that downstream state file upload mutator relies on.
	err := tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	if b.Plan.IsEmpty {
		cmdio.LogString(ctx, "No changes to deploy")
	} else {
		cmdio.LogString(ctx, "Deployment complete")
	}
	return nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
