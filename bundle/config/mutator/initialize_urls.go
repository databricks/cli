package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/workspaceurls"
)

type initializeURLs struct{}

// InitializeURLs makes sure the URL field of each resource is configured.
// NOTE: since this depends on an extra API call, this mutator adds some extra
// latency. As such, it should only be used when needed.
// This URL field is used for the output of the 'bundle summary' CLI command.
func InitializeURLs() bundle.Mutator {
	return &initializeURLs{}
}

func (m *initializeURLs) Name() string {
	return "InitializeURLs"
}

func (m *initializeURLs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	workspaceID, err := b.WorkspaceClient().CurrentWorkspaceID(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	host := b.WorkspaceClient().Config.CanonicalHostName()
	err = initializeForWorkspace(b, workspaceID, host)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func initializeForWorkspace(b *bundle.Bundle, workspaceID int64, host string) error {
	baseURL, err := workspaceurls.WorkspaceBaseURL(host, workspaceID)
	if err != nil {
		return err
	}

	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			r.InitializeURL(*baseURL)
		}
	}

	return nil
}
