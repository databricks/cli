package permissions

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
)

type validateSharedRootPermissions struct{}

func ValidateSharedRootPermissions() bundle.Mutator {
	return &validateSharedRootPermissions{}
}

func (*validateSharedRootPermissions) Name() string {
	return "ValidateSharedRootPermissions"
}

func (*validateSharedRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if libraries.IsWorkspaceSharedPath(b.Config.Workspace.RootPath) {
		return isUsersGroupPermissionSet(b)
	}

	return nil
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
