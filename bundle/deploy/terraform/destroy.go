package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type destroy struct{}

func (w *destroy) Name() string {
	return "terraform.Destroy"
}

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) error {
	// return early if plan is empty
	if b.Plan.IsEmpty {
		cmdio.LogString(ctx, "No resources to destroy in plan. Skipping destroy!")
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return err
	}

	// print the resources that will be destroyed
	err = printPlanSummary(ctx, plan)
	if err != nil {
		return err
	}

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		red := color.New(color.FgRed).SprintFunc()
		b.Plan.ConfirmApply, err = cmdio.AskYesOrNo(ctx, fmt.Sprintf("This will permanently %s resources! Proceed?", red("destroy")))
		if err != nil {
			return err
		}
	}

	// return if confirmation was not provided
	if !b.Plan.ConfirmApply {
		return nil
	}

	if b.Plan.Path == "" {
		return fmt.Errorf("no plan found")
	}

	cmdio.LogString(ctx, "Destroying resources")

	// Apply terraform according to the computed destroy plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}

	cmdio.LogString(ctx, "Destruction complete")
	return nil
}

// Destroy returns a [bundle.Mutator] that runs the conceptual equivalent of
// `terraform destroy ./plan` from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
