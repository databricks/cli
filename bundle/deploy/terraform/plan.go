package terraform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type PlanGoal string

var (
	PlanDeploy  = PlanGoal("deploy")
	PlanDestroy = PlanGoal("destroy")
)

type plan struct {
	goal PlanGoal
}

func (p *plan) Name() string {
	return "terraform.Plan"
}

func (p *plan) Apply(ctx context.Context, b *bundle.Bundle) error {
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	cmdio.LogString(ctx, "Starting plan computation")

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	// Persist computed plan
	tfDir, err := Dir(ctx, b)
	if err != nil {
		return err
	}
	planPath := filepath.Join(tfDir, "plan")
	destroy := p.goal == PlanDestroy

	notEmpty, err := tf.Plan(ctx, tfexec.Destroy(destroy), tfexec.Out(planPath))
	if err != nil {
		return err
	}

	// Set plan in main bundle struct for downstream mutators
	b.Plan = &terraform.Plan{
		Path:         planPath,
		ConfirmApply: b.AutoApprove,
		IsEmpty:      !notEmpty,
	}

	cmdio.LogString(ctx, fmt.Sprintf("Planning complete and persisted at %s\n", planPath))
	return nil
}

// Plan returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func Plan(goal PlanGoal) bundle.Mutator {
	return &plan{
		goal: goal,
	}
}
