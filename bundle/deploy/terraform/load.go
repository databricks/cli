package terraform

import (
	"context"
	"errors"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
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
		return diag.Errorf("terraform init: %v", err)
	}

	state, err := ParseResourcesState(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	err = l.validateState(state)
	if err != nil {
		return diag.FromErr(err)
	}

	// Merge state into configuration.
	err = TerraformToBundle(ctx, state, &b.Config)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func (l *load) validateState(state map[string]map[string]ExportedStateAttributes) error {
	if len(state) == 0 && slices.Contains(l.modes, ErrorOnEmptyState) {
		return errors.New("no deployment state. Did you forget to run 'databricks bundle deploy'?")
	}

	return nil
}

func Load(modes ...loadMode) bundle.Mutator {
	return &load{modes: modes}
}
