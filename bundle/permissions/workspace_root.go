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

type validateSharedRootPermissions struct {
}

func ApplyWorkspaceRootPermissions() bundle.Mutator {
	return &workspaceRootPermissions{}
}

func (*workspaceRootPermissions) Name() string {
	return "ApplyWorkspaceRootPermissions"
}

func ValidateSharedRootPermissions() bundle.Mutator {
	return &validateSharedRootPermissions{}
}

func (*validateSharedRootPermissions) Name() string {
	return "ValidateSharedRootPermissions"
}

// Apply implements bundle.Mutator.
func (*workspaceRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := giveAccessForWorkspaceRoot(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func (*validateSharedRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if isWorkspaceSharedRoot(b.Config.Workspace.RootPath) {
		return isUsersGroupPermissionSet(b)
	}

	return nil
}

func giveAccessForWorkspaceRoot(ctx context.Context, b *bundle.Bundle) error {
	permissions := make([]workspace.WorkspaceObjectAccessControlRequest, 0)

	for _, p := range b.Config.Permissions {
		level, err := getWorkspaceObjectPermissionLevel(p.Level)
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
	obj, err := w.GetStatusByPath(ctx, b.Config.Workspace.RootPath)
	if err != nil {
		return err
	}

	_, err = w.UpdatePermissions(ctx, workspace.WorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   fmt.Sprint(obj.ObjectId),
		WorkspaceObjectType: "directories",
		AccessControlList:   permissions,
	})
	return err
}

func getWorkspaceObjectPermissionLevel(bundlePermission string) (workspace.WorkspaceObjectPermissionLevel, error) {
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

func isWorkspaceSharedRoot(path string) bool {
	return strings.HasPrefix(path, "/Workspace/Shared/")
}

// isUsersGroupPermissionSet checks that top-level permissions set for bundle contain group_name: users with CAN_MANAGE permission.
func isUsersGroupPermissionSet(b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	allUsers := false
	for _, p := range b.Config.Permissions {
		if p.GroupName == "users" && p.Level == CAN_MANAGE {
			allUsers = true
			break
		}
	}

	if !allUsers {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("the bundle root path %s is writable by all workspace users", b.Config.Workspace.RootPath),
			Detail:   "The bundle is configured to use /Workspace/Shared, which will give read/write access to all users. If this is intentional, add CAN_MANAGE for 'group_name: users' permission to your bundle configuration. If the deployment should be restricted, move it to a restricted folder such as /Workspace/Users/<username or principal name>.",
		})
	}

	return diags
}
