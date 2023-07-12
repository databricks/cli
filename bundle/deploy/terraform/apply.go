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
	// Return early if plan is empty
	if b.Plan.IsEmpty {
		cmdio.LogString(ctx, "Terraform plan is empty. Skipping apply.")
		return nil
	}

	// Error if terraform is not initialized
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	// Error if plan is missing
	if b.Plan.Path == "" {
		return fmt.Errorf("no plan found")
	}

	// Read and log plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return err
	}
	err = logPlan(ctx, plan)
	if err != nil {
		return err
	}

	// We do not block for confirmation checks at deploy
	cmdio.LogString(ctx, "Starting deployment")

	// Apply terraform according to the plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	cmdio.LogString(ctx, "Resource deployment completed!")
	return nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
