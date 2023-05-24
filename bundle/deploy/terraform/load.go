package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type load struct{}

func (l *load) Name() string {
	return "terraform.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) error {
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	state, err := b.Terraform.Show(ctx)
	if err != nil {
		return err
	}

	err = ValidateState(state)
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

func ValidateState(state *tfjson.State) error {
	if state.Values == nil {
		return fmt.Errorf("no deployment state. Did you forget to run 'databricks bundle deploy'?")
	}

	if state.Values.RootModule == nil {
		return fmt.Errorf("malformed terraform state: RootModule not set")
	}

	return nil
}

func Load() bundle.Mutator {
	return &load{}
}
