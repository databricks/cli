package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
)

type validateWorkspaceHost struct{}

func ValidateWorkspaceHost() *validateWorkspaceHost {
	return &validateWorkspaceHost{}
}

func (m *validateWorkspaceHost) Name() string {
	return "ValidateWorkspaceHost"
}

func (m *validateWorkspaceHost) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	env_host, _ := env.Host(ctx)
	target_host := b.Config.Workspace.Host

	if env_host != "" && target_host != "" && env_host != target_host {
		return diag.Errorf("target host and DATABRICKS_HOST environment variable mismatch")
	}

	return nil
}
