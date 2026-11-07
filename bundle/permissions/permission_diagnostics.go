package permissions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/set"
)

type permissionDiagnostics struct{}

const (
	CAN_MANAGE = "CAN_MANAGE"
	CAN_VIEW   = "CAN_VIEW"
	CAN_RUN    = "CAN_RUN"
)

func PermissionDiagnostics() bundle.Mutator {
	return &permissionDiagnostics{}
}

func (m *permissionDiagnostics) Name() string {
	return "CheckPermissions"
}

func (m *permissionDiagnostics) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := runAsAccessDiagnostics(b)
	// Only recommend deployment-identity access when top-level permissions are explicit.
	if len(b.Config.Permissions) == 0 {
		return diags
	}

	canManageBundle, _ := analyzeBundlePermissions(b)
	if canManageBundle {
		return diags
	}

	me := b.Config.Workspace.CurrentUser.User
	identityType := "user_name"
	if iamutil.IsServicePrincipal(me) {
		identityType = "service_principal_name"
	}

	return diags.Append(diag.Diagnostic{
		Severity: diag.Recommendation,
		Summary: fmt.Sprintf("permissions section should explicitly include the current deployment identity '%s' or one of its groups\n"+
			"If it is not included, CAN_MANAGE permissions are only applied if the present identity is used to deploy.\n\n"+
			"Consider using a adding a top-level permissions section such as the following:\n\n"+
			"  permissions:\n"+
			"    - %s: %s\n"+
			"      level: CAN_MANAGE\n\n"+
			"See https://docs.databricks.com/dev-tools/bundles/permissions.html to learn more about permission configuration.",
			b.Config.Workspace.CurrentUser.UserName,
			identityType,
			b.Config.Workspace.CurrentUser.UserName,
		),
		Locations: []diag.Location{b.Config.GetLocation("permissions")},
		ID:        diag.PermissionNotIncluded,
	})
}

func runAsAccessDiagnostics(b *bundle.Bundle) diag.Diagnostics {
	runAs := b.Config.RunAs
	if runAs == nil {
		return nil
	}

	identityType, identity := "user_name", runAs.UserName
	if runAs.ServicePrincipalName != "" {
		identityType, identity = "service_principal_name", runAs.ServicePrincipalName
	} else if runAs.GroupName != "" {
		identityType, identity = "group_name", runAs.GroupName
	}

	if identityType != "group_name" && identity == b.Config.Workspace.CurrentUser.UserName {
		return nil
	}

	if b.IsSourceLinkedDeployment() {
		return diag.Diagnostics{{
			Severity:  diag.Warning,
			Summary:   fmt.Sprintf("run_as identity %q may not be able to read source-linked files", identity),
			Detail:    "Source-linked deployments read files from the source workspace folder. Grant the run_as identity CAN_READ access to that folder; top-level bundle permissions do not manage source-linked files.",
			Locations: []diag.Location{b.Config.GetLocation("run_as")},
		}}
	}
	if libraries.IsWorkspaceSharedPath(b.Config.Workspace.FilePath) {
		return diag.Diagnostics{{
			Severity:  diag.Warning,
			Summary:   fmt.Sprintf("run_as identity %q may not be able to read files in /Workspace/Shared", identity),
			Detail:    fmt.Sprintf("Check that the run_as identity has CAN_READ access to %s. Top-level bundle permissions do not manage files in /Workspace/Shared.", b.Config.Workspace.FilePath),
			Locations: []diag.Location{b.Config.GetLocation("run_as")},
		}}
	}

	runAsPermission := resources.Permission{
		UserName:             runAs.UserName,
		ServicePrincipalName: runAs.ServicePrincipalName,
		GroupName:            runAs.GroupName,
	}
	for _, permission := range b.Config.Permissions {
		if samePrincipal(permission, runAsPermission) {
			return nil
		}
	}

	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("run_as identity %q is not in bundle top-level permissions", identity),
		Detail: fmt.Sprintf("Add a top-level permission so this identity can access deployed files and resources:\n"+
			"  permissions:\n"+
			"    - %s: %s\n"+
			"      level: CAN_VIEW", identityType, identity),
		Locations: []diag.Location{b.Config.GetLocation("run_as")},
	}}
}

// analyzeBundlePermissions analyzes the top-level permissions of the bundle.
// This permission set is important since it determines the permissions of the
// target workspace folder.
//
// Returns:
// - canManageBundle: true if the current user or one of their groups can manage the bundle resources.
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
		if p.Level != CAN_MANAGE {
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
		list := otherManagers.Values()
		slices.Sort(list)
		assistance = fmt.Sprintf(
			"For assistance, users or groups with appropriate permissions may include: %s.",
			strings.Join(list, ", "),
		)
	}
	return canManageBundle, assistance
}

func isGroupOfCurrentUser(b *bundle.Bundle, groupName string) bool {
	currentUserGroups := b.Config.Workspace.CurrentUser.Groups

	for _, g := range currentUserGroups {
		if g.Display == groupName {
			return true
		}
	}
	return false
}
