package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type workspaceRootPermissions struct {
}

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
	permissions := make([]workspace.WorkspaceObjectAccessControlRequest, 0)

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
	err := setPermissions(ctx, w, b.Config.Workspace.RootPath, permissions)
	if err != nil {
		return err
	}

	// Adding backslash to the root path
	rootPath := b.Config.Workspace.RootPath
	if rootPath[len(rootPath)-1] != '/' {
		rootPath += "/"
	}

	if !strings.HasPrefix(b.Config.Workspace.ArtifactPath, rootPath) {
		err = setPermissions(ctx, w, b.Config.Workspace.ArtifactPath, permissions)
		if err != nil {
			return err
		}
	}

	if !strings.HasPrefix(b.Config.Workspace.FilePath, rootPath) {
		err = setPermissions(ctx, w, b.Config.Workspace.FilePath, permissions)
		if err != nil {
			return err
		}
	}

	if !strings.HasPrefix(b.Config.Workspace.StatePath, rootPath) {
		err = setPermissions(ctx, w, b.Config.Workspace.StatePath, permissions)
		if err != nil {
			return err
		}
	}

	if !strings.HasPrefix(b.Config.Workspace.ResourcePath, rootPath) {
		err = setPermissions(ctx, w, b.Config.Workspace.ResourcePath, permissions)
		if err != nil {
			return err
		}
	}

	return err
}

func setPermissions(ctx context.Context, w workspace.WorkspaceInterface, path string, permissions []workspace.WorkspaceObjectAccessControlRequest) error {
	obj, err := w.GetStatusByPath(ctx, path)
	if err != nil {
		return err
	}

	_, err = w.SetPermissions(ctx, workspace.WorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   fmt.Sprint(obj.ObjectId),
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
