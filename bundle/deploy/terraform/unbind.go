package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type unbind struct {
	bundleType     string
	tfResourceType string
	resourceKey    string
}

func (m *unbind) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}

	err = tf.StateRm(ctx, fmt.Sprintf("%s.%s", m.tfResourceType, m.resourceKey))
	if err != nil {
		return diag.Errorf("terraform state rm: %v", err)
	}

	// Also remove the permission if it exists
	// First check terraform state list to see if the permission exists
	state, err := tf.Show(ctx)
	if err != nil {
		return diag.Errorf("terraform show: %v", err)
	}

	if state.Values == nil || state.Values.RootModule == nil || state.Values.RootModule.Resources == nil {
		return nil
	}

	for _, resource := range state.Values.RootModule.Resources {
		if resource.Address == fmt.Sprintf("databricks_permissions.%s_%s", m.bundleType, m.resourceKey) {
			err = tf.StateRm(ctx, resource.Address)
			if err != nil {
				return diag.Errorf("terraform state rm: %v", err)
			}
		}
	}

	return nil
}

func (*unbind) Name() string {
	return "terraform.Unbind"
}

func Unbind(bundleType, tfResourceType, resourceKey string) bundle.Mutator {
	return &unbind{bundleType: bundleType, tfResourceType: tfResourceType, resourceKey: resourceKey}
}
