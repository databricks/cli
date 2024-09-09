package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/set"
)

type permissionDiagnostics struct{}

func PermissionDiagnostics() bundle.Mutator {
	return &permissionDiagnostics{}
}

func (m *permissionDiagnostics) Name() string {
	return "CheckPermissions"
}

func (m *permissionDiagnostics) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Permissions) == 0 {
		// Only warn if there is an explicit top-level permissions section
		return nil
	}

	canManageBundle, _ := analyzeBundlePermissions(b)
	if canManageBundle {
		return nil
	}

	return diag.Diagnostics{{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("permissions section should include %s or one of their groups with CAN_MANAGE permissions", b.Config.Workspace.CurrentUser.UserName),
		Locations: []dyn.Location{b.Config.GetLocation("permissions")},
		ID:        diag.PermissionNotIncluded,
	}}
}

// analyzeBundlePermissions analyzes the top-level permissions of the bundle.
// This permission set is important since it determines the permissions of the
// target workspace folder.
//
// Returns:
// - isManager: true if the current user is can manage the bundle resources.
// - assistance: advice on who to contact as to manage this project
func analyzeBundlePermissions(b *bundle.Bundle) (bool, string) {
	canManageBundle := false
	otherManagers := set.NewSet[string]()
	if b.Config.RunAs != nil && b.Config.RunAs.UserName != "" && b.Config.RunAs.UserName != b.Config.Workspace.CurrentUser.UserName {
		// The run_as user is another human that could be contacted
		// about this bundle.
		otherManagers.Add(b.Config.RunAs.UserName)
	}

	currentUser := b.Config.Workspace.CurrentUser.UserName
	targetPermissions := b.Config.Permissions
	for _, p := range targetPermissions {
		if p.Level != CAN_MANAGE && p.Level != IS_OWNER {
			continue
		}

		if p.UserName == currentUser || p.ServicePrincipalName == currentUser {
			canManageBundle = true
			continue
		}

		if isGroupOfCurrentUser(b, p.GroupName) {
			canManageBundle = true
			continue
		}

		// Permission doesn't apply to current user; add to otherManagers
		otherManager := p.UserName
		if otherManager == "" {
			otherManager = p.GroupName
		}
		if otherManager == "" {
			// Skip service principals
			continue
		}
		otherManagers.Add(otherManager)
	}

	assistance := "For assistance, contact the owners of this project."
	if otherManagers.Size() > 0 {
		assistance = fmt.Sprintf(
			"For assistance, users or groups with appropriate permissions may include: %s.",
			strings.Join(otherManagers.Values(), ", "),
		)
	}
	return canManageBundle, assistance
}

func isGroupOfCurrentUser(b *bundle.Bundle, groupName string) bool {
	currentUserGroups := b.Config.Workspace.CurrentUser.User.Groups

	for _, g := range currentUserGroups {
		if g.Display == groupName {
			return true
		}
	}
	return false
}

// ReportPossiblePermissionDenied generates a diagnostic message when a permission denied error is encountered.
//
// Note that since the workspace API doesn't always distinguish between permission denied and path errors,
// we must treat this as a "possible permission error". See acquire.go for more about this.
func ReportPossiblePermissionDenied(ctx context.Context, b *bundle.Bundle, path string) diag.Diagnostics {
	log.Errorf(ctx, "Failed to update, encountered possible permission error: %v", path)

	user := b.Config.Workspace.CurrentUser.DisplayName
	if user == "" {
		user = b.Config.Workspace.CurrentUser.UserName
	}
	canManageBundle, assistance := analyzeBundlePermissions(b)

	if !canManageBundle {
		return diag.Diagnostics{{
			Summary: fmt.Sprintf("unable to deploy to %s as %s.\n"+
				"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n"+
				"%s\n"+
				"They may need to redeploy the bundle to apply the new permissions.\n"+
				"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.",
				path, user, assistance),
			Severity: diag.Error,
			ID:       diag.PathPermissionDenied,
		}}
	}

	// According databricks.yml, the current user has the right permissions.
	// But we're still seeing permission errors. So someone else will need
	// to redeploy the bundle with the right set of permissions.
	return diag.Diagnostics{{
		Summary: fmt.Sprintf("access denied while updating deployment permissions as %s.\n"+
			"%s\n"+
			"They can redeploy the project to apply the latest set of permissions.\n"+
			"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.",
			user, assistance),
		Severity: diag.Error,
		ID:       diag.CannotChangePathPermissions,
	}}
}