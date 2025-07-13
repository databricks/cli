package terraform

import (
	"context"
	"errors"
	"fmt"
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

type planForDestroy struct{}

func (p *planForDestroy) Name() string {
	return "terraform.PlanForDestroy"
}

func (p *planForDestroy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	dir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	plan, err := Plan(ctx, b.Terraform, dir, PlanDestroy)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set plan in main bundle struct for downstream mutators
	b.Plan = plan
	return nil
}

func Plan(
	ctx context.Context,
	tf *tfexec.Terraform,
	tfDir string,
	goal PlanGoal,
) (deployplan.Plan, error) {
	if tf == nil {
		return deployplan.Plan{}, errors.New("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return deployplan.Plan{}, fmt.Errorf("terraform init: %w", err)
	}

	planPath := filepath.Join(tfDir, "plan")
	destroy := goal == PlanDestroy

	notEmpty, err := tf.Plan(ctx, tfexec.Destroy(destroy), tfexec.Out(planPath))
	if err != nil {
		return deployplan.Plan{}, fmt.Errorf("terraform plan: %w", err)
	}

	log.Debugf(ctx, "Planning complete and persisted at %s\n", planPath)
	return deployplan.Plan{
		TerraformPlanPath: planPath,
		TerraformIsEmpty:  !notEmpty,
	}, nil
}

// PlanForDestroy returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func PlanForDestroy() bundle.Mutator {
	return &planForDestroy{}
}
