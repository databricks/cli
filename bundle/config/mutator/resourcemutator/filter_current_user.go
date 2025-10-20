package resourcemutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/iamutil"
)

const (
	isOwner   = "IS_OWNER"
	canManage = "CAN_MANAGE"
)

// This defines which permissions are considered management permissions.
// Current user must have one management permission of themselves; if they don't have any,
// we'll add the first in slice;
// if they end up having both CAN_MANAGE and IS_OWNER, backend may fail with
//
//	Error: cannot create permissions: Permissions being set for UserName([USERNAME]) are ambiguous
//
// Since terraform adds IS_OWNER permission when there is not one, regardless of CAN_MANAGE presence,
// the above error can occur.
// We thus add another bit of logic: we upgrade CAN_MANAGE to IS_OWNER when we can.
var managementPermissions = map[string][]string{
	"jobs":           {isOwner, canManage},
	"pipelines":      {isOwner, canManage},
	"sql_warehouses": {isOwner, canManage},

	// nil means "do nothing"
	"secret_scopes": nil,
}

var defaultManagementPermissions = []string{canManage}

type filterCurrentUser struct{}

// This mutator ensures the current user has the correct permissions for deployed resources.
func EnsureOwnerPermissions() bundle.Mutator {
	return &filterCurrentUser{}
}

func (m *filterCurrentUser) Name() string {
	return "EnsureOwnerPermissions"
}

func ensureCurrentUserPermission(currentUser string) dyn.MapFunc {
	return func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Extract resource type from path: resources.<resource_type>.<resource_name>.permissions
		if len(p) != 4 || p[0].Key() != "resources" || p[3].Key() != "permissions" {
			return v, nil
		}

		resourceType := p[1].Key()

		// Determine the required permission level
		mgmtPerms, ok := managementPermissions[resourceType]
		if !ok {
			mgmtPerms = defaultManagementPermissions
		}

		if len(mgmtPerms) == 0 {
			return v, nil
		}

		// Process permissions array
		return processPermissions(v, currentUser, mgmtPerms)
	}
}

func readUser(v dyn.Value) string {
	userName, _ := dyn.GetValue(v, "user_name").AsString()
	if userName != "" {
		return userName
	}
	servicePrincipalName, _ := dyn.GetValue(v, "service_principal_name").AsString()
	return servicePrincipalName
}

func processPermissions(permissions dyn.Value, currentUser string, mgmtPerms []string) (dyn.Value, error) {
	priorityPermission := mgmtPerms[0]
	permissionToUpgrade := -1

	permissionArray, ok := permissions.AsSequence()
	if !ok {
		return permissions, nil
	}

	for ind, permission := range permissionArray {
		level, ok := dyn.GetValue(permission, "level").AsString()
		if !ok {
			continue
		}
		user := readUser(permission)
		if user == "" || user != currentUser {
			continue
		}
		if level == priorityPermission {
			// already have required permission (IS_OWNER)
			return permissions, nil
		}
		if slices.Contains(mgmtPerms, level) {
			// has management permission that needs to be upgraded to IS_OWNER
			// continue the loop; if we is IS_OWNER we will still bail
			permissionToUpgrade = ind
		}
	}

	if permissionToUpgrade >= 0 {
		v, _ := dyn.Set(permissionArray[permissionToUpgrade], "level", dyn.V(priorityPermission))
		permissionArray[permissionToUpgrade] = v
	} else {
		permissionArray = append(permissionArray, createPermission(currentUser, priorityPermission))
	}

	return dyn.V(permissionArray), nil
}

func createPermission(user, level string) dyn.Value {
	permission := map[string]dyn.Value{
		"level": dyn.V(level),
	}

	// Determine if currentUser is a service principal or user
	if iamutil.IsServicePrincipalName(user) {
		permission["service_principal_name"] = dyn.V(user)
	} else {
		permission["user_name"] = dyn.V(user)
	}

	return dyn.V(permission)
}

func (m *filterCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	currentUser := b.Config.Workspace.CurrentUser.UserName

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		// Use MapByPattern to directly process permissions arrays
		return dyn.MapByPattern(v, dyn.NewPattern(
			dyn.Key("resources"),
			dyn.AnyKey(),
			dyn.AnyKey(),
			dyn.Key("permissions"),
		), ensureCurrentUserPermission(currentUser))
	})

	return diag.FromErr(err)
}
