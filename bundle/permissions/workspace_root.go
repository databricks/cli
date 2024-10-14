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

// Apply implements bundle.Mutator.
func (*workspaceRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {

	if isWorkspaceSharedRoot(b.Config.Workspace.RootPath) {
		diags := checkWorkspaceRootPermissions(b)
		// If there are permissions warnings, return them and do not apply permissions
		// because they are not set correctly for /Workspace/Shared root anyway
		if len(diags) > 0 {
			return diags
		}
	}

	err := giveAccessForWorkspaceRoot(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func (*workspaceRootPermissions) Name() string {
	return "ApplyWorkspaceRootPermissions"
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

// checkWorkspaceRootPermissions checks that if permissions are set for the workspace root, and workspace root starts with /Workspace/Shared, then permissions should be set for group: users
func checkWorkspaceRootPermissions(b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	allUsers := false
	for _, p := range b.Config.Permissions {
		if p.GroupName == "users" && p.Level == CAN_MANAGE {
			allUsers = true
		}
	}

	if !allUsers {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("the bundle root path %s is writable by all workspace users", b.Config.Workspace.RootPath),
			Detail:   "bundle is configured to /Workspace/Shared, which will give read/write access to all users. If all users should have access, add CAN_MANAGE for 'group_name: users' permission to your bundle configuration. If the deployment should be restricted, move it to a restricted folder such as /Users/<username or principal name>",
		})
	}

	return diags
}
