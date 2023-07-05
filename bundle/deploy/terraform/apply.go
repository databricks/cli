package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type ApplyGoal string

const (
	ApplyGoalDeploy  = "deploy"
	ApplyGoalDestroy = "destroy"
)

type apply struct {
	goal ApplyGoal
}

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

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		b.Plan.ConfirmApply, err = cmdio.Ask(ctx, "Proceed with apply? [y/n]: ")
		if err != nil {
			return err
		}
	}
	if !b.Plan.ConfirmApply {
		// return if confirmation was not provided
		return nil
	}

	switch w.goal {
	case ApplyGoalDeploy:
		cmdio.LogString(ctx, "Starting deployment")
	case ApplyGoalDestroy:
		cmdio.LogString(ctx, "Starting destruction")
	}

	// Apply terraform according to the plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	switch w.goal {
	case ApplyGoalDeploy:
		cmdio.LogString(ctx, "Deployment Successful!")
	case ApplyGoalDestroy:
		cmdio.LogString(ctx, "Destruction Successful!")
	}

	return nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply ./plan`
// from the bundle's ephemeral working directory for Terraform.
func Apply(goal ApplyGoal) bundle.Mutator {
	return &apply{goal}
}
