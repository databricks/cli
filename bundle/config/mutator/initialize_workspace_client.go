package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
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
func (m *initializeWorkspaceClient) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if err := validateWorkspaceHost(ctx, b); err != nil {
		return err
	}

	_, err := b.InitializeWorkspaceClient()
	return diag.FromErr(err)
}

func validateWorkspaceHost(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	env_host, _ := env.Host(ctx)
	target_host := b.Config.Workspace.Host

	if env_host != "" && target_host != "" && env_host != target_host {
		return diag.Errorf("target host and DATABRICKS_HOST environment variable mismatch")
	}

	return nil
}
