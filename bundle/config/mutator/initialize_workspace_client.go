package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type initializeWorkspaceClient struct{}

func InitializeWorkspaceClient() bundle.Mutator {
	return &initializeWorkspaceClient{}
}

func (m *initializeWorkspaceClient) Name() string {
	return "InitializeWorkspaceClient"
}

// Apply initializes the workspace client for the bundle. We do this here so
// downstream calls to b.WorkspaceClient() do not panic if there's an error in the
// auth configuration.
func (m *initializeWorkspaceClient) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	_, err := b.InitializeWorkspaceClient()
	return diag.FromErr(err)
}
