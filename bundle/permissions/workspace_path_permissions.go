package permissions

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type WorkspacePathPermissions struct {
	Path        string
	Permissions []resources.Permission
}

func ObjectAclToResourcePermissions(path string, acl []workspace.WorkspaceObjectAccessControlResponse) *WorkspacePathPermissions {
	var permissions []resources.Permission
	for _, a := range acl {
		// Skip the admin group because it's added to all resources by default.
		if a.GroupName == "admins" {
			continue
		}

		for _, pl := range a.AllPermissions {
			permissions = append(permissions, resources.Permission{
				Level:                convertWorkspaceObjectPermissionLevel(pl.PermissionLevel),
				GroupName:            a.GroupName,
				UserName:             a.UserName,
				ServicePrincipalName: a.ServicePrincipalName,
			})
		}
	}

	return &WorkspacePathPermissions{Permissions: permissions, Path: path}
}

func (p WorkspacePathPermissions) Compare(perms []resources.Permission) diag.Diagnostics {
	var diags diag.Diagnostics

	// Check the permissions in the workspace and see if they are all set in the bundle.
	ok, missing := containsAll(p.Permissions, perms)
	if !ok {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "untracked permissions apply to target workspace path",
			Detail:   fmt.Sprintf("The following permissions apply to the workspace folder at %q but are not configured in the bundle:\n%s", p.Path, toString(missing)),
		})
	}

	return diags
}

// containsAll checks if permA contains all permissions in permB.
func containsAll(permA, permB []resources.Permission) (bool, []resources.Permission) {
	var missing []resources.Permission
	for _, a := range permA {
		found := false
		for _, b := range permB {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, a)
		}
	}
	return len(missing) == 0, missing
}

// convertWorkspaceObjectPermissionLevel converts matching object permission levels to bundle ones.
// If there is no matching permission level, it returns permission level as is, for example, CAN_EDIT.
func convertWorkspaceObjectPermissionLevel(level workspace.WorkspaceObjectPermissionLevel) string {
	switch level {
	case workspace.WorkspaceObjectPermissionLevelCanRead:
		return CAN_VIEW
	default:
		return string(level)
	}
}

func toString(p []resources.Permission) string {
	var sb strings.Builder
	for _, perm := range p {
		sb.WriteString(fmt.Sprintf("- %s\n", perm.String()))
	}
	return sb.String()
}
