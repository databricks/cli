package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/plan"
)

type planDeploy struct{}

func (p *planDeploy) Name() string {
	return "terraform.PlanDeploy"
}

// TODO: We need end to end tests for the approval flow.
func (p *planDeploy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	// Get terraform project root.
	tfDir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set plan in main bundle struct for downstream mutators
	b.Plan, err = plan.NewTerraformPlan(ctx, plan.TerraformPlanOpts{
		Executable: tf,
		Root:       tfDir,
		IsDestroy:  false,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// If the `--auto-approve` flag is set, we don't need to ask for approval.
	if b.AutoApprove {
		return nil
	}

	destructiveActions := b.Plan.ActionsByTypes(plan.ActionTypeRecreate, plan.ActionTypeDelete)

	// If there are no deletes or recreates, we don't need to ask for approval.
	if len(destructiveActions) == 0 {
		return nil
	}

	// display information the user needs to know about the deploy plan.
	cmdio.LogString(ctx, "The following resources will be deleted or recreated:")
	for _, a := range destructiveActions {
		cmdio.Log(ctx, a)
	}

	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return diag.FromErr(err)
	}

	// TODO: Remove the previous flag for deployment in the bundle tree?
	if !approved {
		// We error here to terminate the control flow and prevent the current
		// process from modifying any deployment state.
		cmdio.LogString(ctx, "No changes are being made...")
		return diag.FromErr(bundle.ErrorBreakSequence)
	}

	return nil
}

// PlanDeploy returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func PlanDeploy() bundle.Mutator {
	return &planDeploy{}
}
