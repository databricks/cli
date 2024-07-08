package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// TODO: Release lock on error?
type tfDeploy struct{}

func (w *tfDeploy) Name() string {
	return "terraform.Deploy"
}

// TODO: Document in PR why the approval logic is being moved to the plan mutator.
func (w *tfDeploy) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// return early if plan is empty
	if b.Plan.IsEmpty() {
		cmdio.LogString(ctx, "No changes to deploy...")
		return nil
	}

	cmdio.LogString(ctx, "Deploying resources...")

	err := b.Plan.Apply()
	if err != nil {
		return diag.Errorf("terraform apply: %v", err)
	}

	log.Infof(ctx, "Resource deployment completed")
	return nil
}

// Deploy returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Deploy() bundle.Mutator {
	return &tfDeploy{}
}
