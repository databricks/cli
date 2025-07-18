package deploy

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func ResourcePathMkdir() bundle.Mutator {
	return &resourcePathMkdir{}
}

type resourcePathMkdir struct{}

func (m *resourcePathMkdir) Name() string {
	return "deploy:resource_path_mkdir"
}

func (m *resourcePathMkdir) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Only dashboards and alerts need ${workspace.resource_path} to exist. They are
	// created at this node in the workspace file tree.
	if len(b.Config.Resources.Dashboards) == 0 && len(b.Config.Resources.Alerts) == 0 {
		return nil
	}

	w := b.WorkspaceClient()

	// Create the resource path if it doesn't exist.
	var aerr *apierr.APIError
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.ResourcePath)
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		err := w.Workspace.MkdirsByPath(ctx, b.Config.Workspace.ResourcePath)
		return diag.FromErr(err)
	}
	return diag.FromErr(err)
}
