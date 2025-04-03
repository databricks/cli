package permissions

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/paths"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type workspaceRootPermissions struct{}

func ApplyWorkspaceRootPermissions() bundle.Mutator {
	return &workspaceRootPermissions{}
}

func (*workspaceRootPermissions) Name() string {
	return "ApplyWorkspaceRootPermissions"
}

// Apply implements bundle.Mutator.
func (*workspaceRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := giveAccessForWorkspaceRoot(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func giveAccessForWorkspaceRoot(ctx context.Context, b *bundle.Bundle) error {
	var permissions []workspace.WorkspaceObjectAccessControlRequest

	for _, p := range b.Config.Permissions {
		level, err := GetWorkspaceObjectPermissionLevel(p.Level)
		if err != nil {
			return err
		}

		permissions = append(permissions, workspace.WorkspaceObjectAccessControlRequest{
			GroupName:            p.GroupName,
			UserName:             p.UserName,
			ServicePrincipalName: p.ServicePrincipalName,
			PermissionLevel:      level,
		})
	}

	if len(permissions) == 0 {
		return nil
	}

	w := b.WorkspaceClient().Workspace
	bundlePaths := paths.CollectUniqueWorkspacePathPrefixes(b.Config.Workspace)

	g, ctx := errgroup.WithContext(ctx)
	for _, p := range bundlePaths {
		g.Go(func() error {
			return setPermissions(ctx, w, p, permissions)
		})
	}

	return g.Wait()
}

func setPermissions(ctx context.Context, w workspace.WorkspaceInterface, path string, permissions []workspace.WorkspaceObjectAccessControlRequest) error {
	// If the folder is shared, then we don't need to set permissions since it's always set for all users and it's checked in mutators before.
	if libraries.IsWorkspaceSharedPath(path) {
		return nil
	}

	obj, err := w.GetStatusByPath(ctx, path)
	if err != nil {
		return err
	}

	_, err = w.SetPermissions(ctx, workspace.WorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   strconv.FormatInt(obj.ObjectId, 10),
		WorkspaceObjectType: "directories",
		AccessControlList:   permissions,
	})

	return err
}

func GetWorkspaceObjectPermissionLevel(bundlePermission string) (workspace.WorkspaceObjectPermissionLevel, error) {
	switch bundlePermission {
	case CAN_MANAGE:
		return workspace.WorkspaceObjectPermissionLevelCanManage, nil
	case CAN_RUN:
		return workspace.WorkspaceObjectPermissionLevelCanRun, nil
	case CAN_VIEW:
		return workspace.WorkspaceObjectPermissionLevelCanRead, nil
	default:
		return "", fmt.Errorf("unsupported bundle permission level %s", bundlePermission)
	}
}
