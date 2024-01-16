package terraform

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type loadMode int

const ErrorOnEmptyState loadMode = 0
const ReplaceResources loadMode = 1

type load struct {
	modes []loadMode
}

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

	err = l.validateState(state)
	if err != nil {
		return err
	}

	if slices.Contains(l.modes, ReplaceResources) {
		b.Config.Resources = config.Resources{}
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
			return fmt.Errorf("no deployment state. Did you forget to run 'databricks bundle deploy'?")
		}
		return nil
	}

	if state.Values.RootModule == nil {
		return fmt.Errorf("malformed terraform state: RootModule not set")
	}

	return nil
}

func Load(modes ...loadMode) bundle.Mutator {
	return &load{modes: modes}
}
