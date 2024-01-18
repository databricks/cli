package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type importResource struct {
	resourceType string
	resourceKey  string
	resourceId   string
}

// Apply implements bundle.Mutator.
func (m *importResource) Apply(ctx context.Context, b *bundle.Bundle) error {
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	return tf.Import(ctx, fmt.Sprintf("%s.%s", m.resourceType, m.resourceKey), m.resourceId)
}

// Name implements bundle.Mutator.
func (*importResource) Name() string {
	return "terraform.Import"
}

func Import(resourceType string, resourceKey string, resourceId string) bundle.Mutator {
	return &importResource{resourceType: resourceType, resourceKey: resourceKey, resourceId: resourceId}
}
