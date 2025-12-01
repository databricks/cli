package terraform

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
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
	diags := Initialize(ctx, b)
	if diags.HasError() {
		return diags
	}

	// Persist computed plan
	tfDir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	planPath := filepath.Join(tfDir, "plan")
	destroy := p.goal == PlanDestroy

	notEmpty, err := b.Terraform.Plan(ctx, tfexec.Destroy(destroy), tfexec.Out(planPath))
	if err != nil {
		return diag.FromErr(err)
	}

	// Set plan in main bundle struct for downstream mutators
	b.TerraformPlanPath = planPath
	b.TerraformPlanIsEmpty = !notEmpty

	log.Debugf(ctx, "Planning complete and persisted at %s\n", planPath)
	return diags
}

// Plan returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func Plan(goal PlanGoal) bundle.Mutator {
	return &plan{
		goal: goal,
	}
}
