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
	permissions := make([]resources.Permission, 0)
	for _, a := range acl {
		// Skip the admin group because it's added to all resources by default.
		if a.GroupName == "admin" {
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

	// Check the permissions in the bundle and see if they are all set in the workspace.
	ok, missing := containsAll(perms, p.Permissions)
	if !ok {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "permissions missing",
			Detail:   fmt.Sprintf("Following permissions set in the bundle but not set for workspace folder %s:\n%s", p.Path, toString(missing)),
		})
	}

	// Check the permissions in the workspace and see if they are all set in the bundle.
	ok, missing = containsAll(p.Permissions, perms)
	if !ok {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "permissions missing",
			Detail:   fmt.Sprintf("Following permissions set for the workspace folder but not set for bundle %s:\n%s", p.Path, toString(missing)),
		})
	}

	return diags
}

// containsAll checks if permA contains all permissions in permB.
func containsAll(permA []resources.Permission, permB []resources.Permission) (bool, []resources.Permission) {
	missing := make([]resources.Permission, 0)
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
		if perm.ServicePrincipalName != "" {
			sb.WriteString(fmt.Sprintf("- level: %s\n  service_principal_name: %s\n", perm.Level, perm.ServicePrincipalName))
			continue
		}

		if perm.GroupName != "" {
			sb.WriteString(fmt.Sprintf("- level: %s\n  group_name: %s\n", perm.Level, perm.GroupName))
			continue
		}

		if perm.UserName != "" {
			sb.WriteString(fmt.Sprintf("- level: %s\n  user_name: %s\n", perm.Level, perm.UserName))
			continue
		}
	}
	return sb.String()
}
