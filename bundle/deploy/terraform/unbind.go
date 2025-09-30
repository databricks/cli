package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type unbind struct {
	resourceType string
	resourceKey  string
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

	err = tf.StateRm(ctx, fmt.Sprintf("%s.%s", m.resourceType, m.resourceKey))
	if err != nil {
		return diag.Errorf("terraform state rm: %v", err)
	}

	return nil
}

func (*unbind) Name() string {
	return "terraform.Unbind"
}

func Unbind(resourceType, resourceKey string) bundle.Mutator {
	return &unbind{resourceType: resourceType, resourceKey: resourceKey}
}
