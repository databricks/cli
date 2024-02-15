package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type unbind struct {
	resourceType string
	resourceKey  string
}

func (m *unbind) Apply(ctx context.Context, b *bundle.Bundle) error {
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	err = tf.StateRm(ctx, fmt.Sprintf("%s.%s", m.resourceType, m.resourceKey))
	if err != nil {
		return fmt.Errorf("terraform state rm: %w", err)
	}

	return nil
}

func (*unbind) Name() string {
	return "terraform.Unbind"
}

func Unbind(resourceType string, resourceKey string) bundle.Mutator {
	return &unbind{resourceType: resourceType, resourceKey: resourceKey}
}
