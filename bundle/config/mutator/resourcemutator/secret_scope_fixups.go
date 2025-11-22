package resourcemutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type secretScopeFixups struct {
	engine engine.EngineType
}

func SecretScopeFixups(engine engine.EngineType) bundle.Mutator {
	return &secretScopeFixups{engine: engine}
}

func (m *secretScopeFixups) Name() string {
	return "SecretScopeFixups"
}

func addManageForCurrentUser(scope *resources.SecretScope, currentUser *iam.User) {
	// Check if current user already has a permission configured
	for _, perm := range scope.Permissions {
		if (perm.UserName == currentUser.UserName || perm.ServicePrincipalName == currentUser.UserName) && perm.Level == resources.SecretScopePermissionLevelManage {
			return
		}
	}

	acl := resources.SecretScopePermission{
		Level: resources.SecretScopePermissionLevelManage,
	}
	if iamutil.IsServicePrincipal(currentUser) {
		acl.ServicePrincipalName = currentUser.UserName
	} else {
		acl.UserName = currentUser.UserName
	}
	scope.Permissions = append(scope.Permissions, acl)
}

// If a principal has multiple permissions configured, only retain the highest level.
// MANAGE > WRITE > READ.
func collapsePermissions(scope *resources.SecretScope) error {
	// Map to track the highest permission for each principal
	principalPermissions := make(map[string]resources.SecretScopePermission)

	// Define permission hierarchy
	permissionRank := map[resources.SecretScopePermissionLevel]int{
		resources.SecretScopePermissionLevelRead:   1,
		resources.SecretScopePermissionLevelWrite:  2,
		resources.SecretScopePermissionLevelManage: 3,
	}

	// Process all permissions and keep the highest level for each principal
	for _, perm := range scope.Permissions {
		// Validate permission level
		if _, ok := permissionRank[perm.Level]; !ok {
			return fmt.Errorf("unknown permission level %q for secret scope", perm.Level)
		}

		// Add a prefix to retain the original principal type. In practice collisions here should
		// be rare, if any. But adding the prefix is defensive.
		var principal string
		if perm.UserName != "" {
			principal = "user:" + perm.UserName
		} else if perm.GroupName != "" {
			principal = "group:" + perm.GroupName
		} else if perm.ServicePrincipalName != "" {
			principal = "sp:" + perm.ServicePrincipalName
		} else {
			return fmt.Errorf("missing principal in permissions for secret scope %q", scope.Name)
		}

		existing, exists := principalPermissions[principal]
		if !exists || permissionRank[perm.Level] > permissionRank[existing.Level] {
			principalPermissions[principal] = perm
		}
	}

	// Rebuild the permissions list with deduplicated entries
	newPermissions := make([]resources.SecretScopePermission, 0, len(principalPermissions))
	for _, perm := range principalPermissions {
		newPermissions = append(newPermissions, perm)
	}

	slices.SortFunc(newPermissions, func(a, b resources.SecretScopePermission) int {
		var principalA string
		switch {
		case a.UserName != "":
			principalA = "user:" + a.UserName
		case a.ServicePrincipalName != "":
			principalA = "sp:" + a.ServicePrincipalName
		case a.GroupName != "":
			principalA = "group:" + a.GroupName
		}

		var principalB string
		switch {
		case b.UserName != "":
			principalB = "user:" + b.UserName
		case b.ServicePrincipalName != "":
			principalB = "sp:" + b.ServicePrincipalName
		case b.GroupName != "":
			principalB = "group:" + b.GroupName
		}

		return strings.Compare(principalA, principalB)
	})

	scope.Permissions = newPermissions
	return nil
}

func (m *secretScopeFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Secret scopes by default have the current user as a MANAGE ACL. We need to add it to the client ACL list
	// to prevent a phantom persistent diff.
	// We do not need to do this in terraform because terraform naively always applies the config during ACL
	// creation without checking if the ACL already exists.
	// https://github.com/databricks/terraform-provider-databricks/blob/5cb5d3fa46bc4843be1a4c4bce89296eaa2e14fc/secrets/resource_secret_acl.go#L43
	if !m.engine.IsDirect() {
		return nil
	}

	// Secret scopes assigns the create MANAGE ACL on it by default. So we always add it to
	// the client ACL list as a default.
	for key, scope := range b.Config.Resources.SecretScopes {
		if scope == nil {
			continue
		}

		currentUser := b.Config.Workspace.CurrentUser.User

		addManageForCurrentUser(scope, currentUser)
		err := collapsePermissions(scope)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:  diag.Error,
					Summary:   "Failed to collapse permissions for secret scope",
					Detail:    err.Error(),
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.secret_scopes." + key)},
					Locations: []dyn.Location{b.Config.GetLocation("resources.secret_scopes." + key)},
				},
			}
		}
	}

	return nil
}
