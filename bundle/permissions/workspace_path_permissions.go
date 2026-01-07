package permissions

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config"
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

		// Find the highest permission level for this principal (handles inherited + explicit permissions)
		var highestLevel string
		for _, pl := range a.AllPermissions {
			level := convertWorkspaceObjectPermissionLevel(pl.PermissionLevel)
			if resources.GetLevelScore(level) > resources.GetLevelScore(highestLevel) {
				highestLevel = level
			}
		}

		if highestLevel != "" {
			permissions = append(permissions, resources.Permission{
				Level:                highestLevel,
				GroupName:            a.GroupName,
				UserName:             a.UserName,
				ServicePrincipalName: a.ServicePrincipalName,
			})
		}
	}

	return &WorkspacePathPermissions{Permissions: permissions, Path: path}
}

func (p WorkspacePathPermissions) Compare(perms []resources.Permission, currentUser *config.User) diag.Diagnostics {
	var diags diag.Diagnostics

	// Check the permissions in the workspace and see if they are all set in the bundle.
	ok, missing := containsAll(p.Permissions, perms, currentUser)
	if !ok {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "workspace folder has permissions not configured in bundle",
			Detail: fmt.Sprintf(
				"The following permissions apply to the workspace folder at %q "+
					"but are not configured in the bundle:\n%s\n"+
					"Add them to your bundle permissions or remove them from the folder.\n"+
					"See https://docs.databricks.com/dev-tools/bundles/permissions",
				p.Path, toString(missing)),
		})
	}

	return diags
}

// samePrincipal checks if two permissions refer to the same user/group/service principal.
func samePrincipal(a, b resources.Permission) bool {
	return a.UserName == b.UserName &&
		a.GroupName == b.GroupName &&
		a.ServicePrincipalName == b.ServicePrincipalName
}

// containsAll checks if all permissions in permA (workspace) are accounted for in permB (bundle).
// A workspace permission is considered accounted for if the bundle has the same principal
// with an equal or higher permission level, OR if the permission is for a group that belongs
// to the current deployment user and the current user is already tracked in the bundle.
func containsAll(permA, permB []resources.Permission, currentUser *config.User) (bool, []resources.Permission) {
	var missing []resources.Permission
	for _, a := range permA {
		found := false

		// Check if bundle has same principal with adequate permission
		for _, b := range permB {
			if samePrincipal(a, b) && resources.GetLevelScore(b.Level) >= resources.GetLevelScore(a.Level) {
				found = true
				break
			}
		}

		// If not found directly, check if this is a group of the current user
		// and the current user is already tracked in the bundle
		if !found && a.GroupName != "" && currentUser != nil {
			if isCurrentUserGroup(a.GroupName, currentUser) && currentUserInBundle(permB, currentUser) {
				found = true
			}
		}

		if !found {
			missing = append(missing, a)
		}
	}
	return len(missing) == 0, missing
}

func isCurrentUserGroup(groupName string, currentUser *config.User) bool {
	for _, g := range currentUser.Groups {
		// Check both Display (preferred) and Value (fallback) fields
		if g.Display == groupName || g.Value == groupName {
			return true
		}
	}
	return false
}

func currentUserInBundle(perms []resources.Permission, currentUser *config.User) bool {
	for _, p := range perms {
		// Check direct user match
		if p.UserName == currentUser.UserName || p.ServicePrincipalName == currentUser.UserName {
			return true
		}
		// Check if any bundle group contains the current user
		if p.GroupName != "" && isCurrentUserGroup(p.GroupName, currentUser) {
			return true
		}
	}
	return false
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
