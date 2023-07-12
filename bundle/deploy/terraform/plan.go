package terraform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func logPlan(ctx context.Context, plan *tfjson.Plan) error {
	cmdio.LogString(ctx, "Plan:")
	for _, change := range plan.ResourceChanges {
		tfActions := change.Change.Actions
		if tfActions.Read() || tfActions.NoOp() {
			continue
		}

		var action string
		switch {
		case tfActions.Update():
			action = "Update"
		case tfActions.Create():
			action = "Create"
		case tfActions.Delete():
			action = "Delete"
		case tfActions.Replace():
			action = "Replace"
		default:
			return fmt.Errorf("unknown terraform actions: %s", tfActions)
		}

		resourceType := change.Type
		switch resourceType {
		case "databricks_job":
			resourceType = "Job"
		case "databricks_pipeline":
			resourceType = "DLT Pipeline"
		case "databricks_mlflow_model":
			resourceType = "Mlflow Model"
		case "databricks_mlflow_experiment":
			resourceType = "Mlflow Experiment"
		}

		err := cmdio.RenderWithTemplate(ctx, change, fmt.Sprintf("%s %s {{.Name}}\n", action, resourceType))
		if err != nil {
			return err
		}
	}
	cmdio.LogString(ctx, "")
	return nil
}

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
	tfDir, err := Dir(b)
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
