package deploy

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

// needsResourcePath reports whether r declares a resource of a type that needs
// ${workspace.resource_path} to exist.
func needsResourcePath(r *config.Resources) bool {
	for _, group := range r.AllResources() {
		if _, ok := config.ResourceTypeAs[resources.ResourcePathUser](group.Description.PluralName); ok && len(group.Resources) > 0 {
			return true
		}
	}
	return false
}

func (m *resourcePathMkdir) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !needsResourcePath(&b.Config.Resources) {
		return nil
	}

	w := b.WorkspaceClient(ctx)

	// Optimisitcally create the resource path. If it already exists ignore the error.
	err := w.Workspace.MkdirsByPath(ctx, b.Config.Workspace.ResourcePath)
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.ErrorCode == "RESOURCE_ALREADY_EXISTS" {
		return nil
	}
	return diag.FromErr(err)
}
