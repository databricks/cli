package terraform

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
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

func (p *plan) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}

	// Persist computed plan
	tfDir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	planPath := filepath.Join(tfDir, "plan")
	destroy := p.goal == PlanDestroy

	notEmpty, err := tf.Plan(ctx, tfexec.Destroy(destroy), tfexec.Out(planPath))
	if err != nil {
		return diag.FromErr(err)
	}

	// Set plan in main bundle struct for downstream mutators
	b.Plan = deployplan.Plan{
		TerraformPlanPath: planPath,
		TerraformIsEmpty:  !notEmpty,
	}

	log.Debugf(ctx, "Planning complete and persisted at %s\n", planPath)
	return nil
}

// Plan returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func Plan(goal PlanGoal) bundle.Mutator {
	return &plan{
		goal: goal,
	}
}
