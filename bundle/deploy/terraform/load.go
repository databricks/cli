package terraform

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type loadMode int

const ErrorOnEmptyState loadMode = 0

type load struct {
	modes []loadMode
}

func (l *load) Name() string {
	return "terraform.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %w", err)
	}

	state, err := b.Terraform.Show(ctx)
	if err != nil {
		return err
	}

	err = l.validateState(state)
	if err != nil {
		return err
	}

	// Merge state into configuration.
	err = TerraformToBundle(state, &b.Config)
	if err != nil {
		return err
	}

	return nil
}

func (l *load) validateState(state *tfjson.State) error {
	if state.Values == nil {
		if slices.Contains(l.modes, ErrorOnEmptyState) {
			return diag.Errorf("no deployment state. Did you forget to run 'databricks bundle deploy'?")
		}
		return nil
	}

	if state.Values.RootModule == nil {
		return diag.Errorf("malformed terraform state: RootModule not set")
	}

	return nil
}

func Load(modes ...loadMode) bundle.Mutator {
	return &load{modes: modes}
}
