package permissions

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type setOwnerPermissions struct{}

func SetOwnerPermissions() bundle.Mutator {
	return &setOwnerPermissions{}
}

func (m *setOwnerPermissions) Name() string {
	return "SetOwnerPermissions"
}

func (m *setOwnerPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// TODO: set CAN_MANAGE permissions based on the 'owner' property
	return nil
}
