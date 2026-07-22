package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/paths"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"golang.org/x/sync/errgroup"
)

type folderPermissions struct{ bundle.RO }

func (f *folderPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Permissions) == 0 {
		return nil
	}

	bundlePaths := paths.CollectUniqueWorkspacePathPrefixes(b.Config.Workspace).Paths

	var diags diag.Diagnostics
	g, ctx := errgroup.WithContext(ctx)
	results := make([]diag.Diagnostics, len(bundlePaths))
	for i, p := range bundlePaths {
		g.Go(func() error {
			results[i] = checkFolderPermission(ctx, b, p)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Note, only diag from first coroutine is captured, others are lost
		diags = diags.Extend(diag.FromErr(err))
	}

	for _, r := range results {
		diags = diags.Extend(r)
	}

	return diags
}

func checkFolderPermission(ctx context.Context, b *bundle.Bundle, folderPath string) diag.Diagnostics {
	// If the folder is shared, then we don't need to check permissions as it was already checked in the other mutator before.
	if libraries.IsWorkspaceSharedPath(folderPath) {
		return nil
	}

	w := b.WorkspaceClient(ctx).Workspace
	acl, err := permissions.ResolveFolderACL(ctx, w, folderPath)
	if err != nil {
		return diag.FromErr(err)
	}

	return acl.Permissions.Compare(b.Config.Permissions)
}

// Name implements bundle.ReadOnlyMutator.
func (f *folderPermissions) Name() string {
	return "validate:folder_permissions"
}

// ValidateFolderPermissions validates that permissions for the folders in Workspace file system matches
// the permissions in the top-level permissions section of the bundle.
func ValidateFolderPermissions() bundle.ReadOnlyMutator {
	return &folderPermissions{}
}
