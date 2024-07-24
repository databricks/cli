package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/plan"
)

type planDestroy struct{}

func (p *planDestroy) Name() string {
	return "terraform.PlanDestroy"
}

// TODO: We need end to end tests for the approval flow.
func (p *planDestroy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	// Get terraform project root.
	tfDir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Plan, err = plan.NewTerraformPlan(ctx, plan.TerraformPlanOpts{
		Executable: tf,
		Root:       tfDir,
		IsDestroy:  true,
	})

	// TODO: Note this PR also makes destroy idempotent.

	// Display information needs to know about the destroy plan.
	deleteActions := b.Plan.ActionsByTypes(plan.ActionTypeDelete)

	// If there are delete actions, show that information to the user.
	if len(deleteActions) != 0 {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
			cmdio.Log(ctx, a)
		}
		// Log new line for better presentation.
		cmdio.LogString(ctx, "")
	}

	// TODO: write e2e integration tests / unit tests to check this property.
	// TODO: Also test this manually.
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.RootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	// If the root path exists, show warning.
	_, err = f.Stat(ctx, "")
	if err == nil {
		cmdio.LogString(ctx, fmt.Sprintf("All files and directories at %s will be deleted.", b.Config.Workspace.RootPath))
		// Log new line for better presentation.
		cmdio.LogString(ctx, "")
	}

	// If the `--auto-approve` flag is set, we don't need to ask for approval.
	if b.AutoApprove {
		return nil
	}

	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return diag.FromErr(err)
	}

	// TODO: Remove the previous flag for deployment in the bundle tree?
	if !approved {
		cmdio.LogString(ctx, "No changes are being made...")
		return diag.FromErr(bundle.ErrorBreakSequence)
	}

	return nil
}

// Plan returns a [bundle.Mutator] that runs the equivalent of `terraform plan -out ./plan`
// from the bundle's ephemeral working directory for Terraform.
func PlanDestroy() bundle.Mutator {
	return &planDestroy{}
}
