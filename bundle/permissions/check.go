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

func NewFromWorkspaceObjectAcl(path string, acl []workspace.WorkspaceObjectAccessControlResponse) *WorkspacePathPermissions {
	permissions := make([]resources.Permission, 0)
	for _, a := range acl {
		// Skip the admin group because it's added to all resources by default.
		if a.GroupName == "admin" {
			continue
		}

		for _, pl := range a.AllPermissions {
			permissions = append(permissions, resources.Permission{
				Level:                string(pl.PermissionLevel),
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

	if len(p.Permissions) != len(perms) {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "permissions count mismatch",
			Detail: fmt.Sprintf(
				"The number of permissions in the bundle is %d, but the number of permissions in the workspace is %d\n%s\n%s",
				len(perms), len(p.Permissions),
				toString("Bundle permissions", p.Permissions), toString("Workspace permissions", perms)),
		})
		return diags
	}

	for _, perm := range perms {
		level, err := GetWorkspaceObjectPermissionLevel(perm.Level)
		if err != nil {
			return diag.FromErr(err)
		}

		found := false
		for _, objPerm := range p.Permissions {
			if objPerm.GroupName == perm.GroupName &&
				objPerm.UserName == perm.UserName &&
				objPerm.ServicePrincipalName == perm.ServicePrincipalName &&
				objPerm.Level == string(level) {
				found = true
				break
			}
		}

		if !found {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "permission not found",
				Detail: fmt.Sprintf(
					"Permission (%s) not set for bundle workspace folder %s\n%s\n%s",
					perm, p.Path,
					toString("Bundle permissions", p.Permissions), toString("Workspace permissions", perms)),
			})
		}
	}

	return diags
}

func toString(title string, p []resources.Permission) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n", title))
	for _, perm := range p {
		sb.WriteString(fmt.Sprintf("- level: %s, user_name: %s, group_name: %s, service_principal_name: %s\n", perm.Level, perm.UserName, perm.GroupName, perm.ServicePrincipalName))
	}
	return sb.String()
}
