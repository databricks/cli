package validate

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/paths"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type folderPermissions struct{ bundle.RO }

func (f *folderPermissions) Apply(ctx context.Context, b *bundle.Bundle) error {
	if len(b.Config.Permissions) == 0 {
		return nil
	}

	bundlePaths := paths.CollectUniqueWorkspacePathPrefixes(b.Config.Workspace).Paths

	g, ctx := errgroup.WithContext(ctx)
	results := make([]diag.Diagnostics, len(bundlePaths))
	for i, p := range bundlePaths {
		g.Go(func() error {
			diags, err := checkFolderPermission(ctx, b, p)
			if err != nil {
				return err
			}
			results[i] = diags
			return nil
		})
	}

	// Note, only error from first coroutine is captured, others are lost
	if err := g.Wait(); err != nil {
		return err
	}

	for _, r := range results {
		for _, d := range r {
			logdiag.LogDiag(ctx, d)
		}
	}

	return nil
}

func checkFolderPermission(ctx context.Context, b *bundle.Bundle, folderPath string) (diag.Diagnostics, error) {
	// If the folder is shared, then we don't need to check permissions as it was already checked in the other mutator before.
	if libraries.IsWorkspaceSharedPath(folderPath) {
		return nil, nil
	}

	w := b.WorkspaceClient(ctx).Workspace
	obj, err := getClosestExistingObject(ctx, w, folderPath)
	if err != nil {
		return nil, err
	}

	objPermissions, err := w.GetPermissions(ctx, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   strconv.FormatInt(obj.ObjectId, 10),
		WorkspaceObjectType: "directories",
	})
	if err != nil {
		return nil, err
	}

	p := permissions.ObjectAclToResourcePermissions(folderPath, objPermissions.AccessControlList)
	return p.Compare(b.Config.Permissions), nil
}

func getClosestExistingObject(ctx context.Context, w workspace.WorkspaceInterface, folderPath string) (*workspace.ObjectInfo, error) {
	for {
		obj, err := w.GetStatusByPath(ctx, folderPath)
		if err == nil {
			return obj, nil
		}

		if !apierr.IsMissing(err) {
			return nil, err
		}

		parent := path.Dir(folderPath)
		// If the parent is the same as the current folder, then we have reached the root
		if folderPath == parent {
			break
		}

		folderPath = parent
	}

	return nil, fmt.Errorf("folder %s and its parent folders do not exist", folderPath)
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
