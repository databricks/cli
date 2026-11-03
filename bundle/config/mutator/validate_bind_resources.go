package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validateBindResources struct{}

// ValidateBindResources validates that bind blocks only contain valid resource types.
// Binding is only allowed for resources directly under the resources block,
// not for child resources like permissions or grants.
func ValidateBindResources() bundle.Mutator {
	return &validateBindResources{}
}

func (m *validateBindResources) Name() string {
	return "ValidateBindResources"
}

func (m *validateBindResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Target == nil {
		return nil
	}

	return b.Target.Bind.Validate()
}
