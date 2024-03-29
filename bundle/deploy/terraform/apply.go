package terraform

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type apply struct{}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	cmdio.LogString(ctx, "Deploying resources...")

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}

	err = tf.Apply(ctx)
	if err != nil {
		return diag.Errorf("terraform apply: %v", err)
	}

	log.Infof(ctx, "Resource deployment completed")
	return nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply() bundle.Mutator {
	return &apply{}
}
