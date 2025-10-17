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

// defines which permissions are considered management permissions
// user must have one management permission of themselves; if they don't have any,
// we'll add the first in slice
var managementPermissions = map[string][]string{
	"jobs":           {isOwner},
	"pipelines":      {isOwner, canManage},
	"sql_warehouses": {isOwner},

	// nil means "do nothing"
	"secret_scopes": nil,
}

var defaultManagementPermissions = []string{canManage}

type filterCurrentUser struct{}

// This mutator ensures the current user has the correct permissions for deployed resources:
// - For jobs and pipelines: ensures IS_OWNER permission, removes other permissions of current user
// - For other resources: ensures CAN_MANAGE permission, removes other permissions of current user
func FilterCurrentUser() bundle.Mutator {
	return &filterCurrentUser{}
}

func (m *filterCurrentUser) Name() string {
	return "EnsureCurrentUserPermissions"
}

func ensureCurrentUserPermission(currentUser string) dyn.WalkValueFunc {
	return func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// We're looking for resource collections (jobs, pipelines, etc.)
		// at depth 1: [resource_type]
		if len(p) != 1 {
			return v, nil
		}

		resourceType := p[0].Key()

		// Process each resource in the collection using MapByPattern
		return dyn.MapByPattern(v, dyn.NewPattern(dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Get permissions array for this resource
			permissions, err := dyn.Get(v, "permissions")
			if dyn.IsNoSuchKeyError(err) {
				// No permissions defined, leave it alone
				return v, nil
			}
			if err != nil {
				return dyn.InvalidValue, err
			}

			// Determine the required permission level
			mgmtPerms, ok := managementPermissions[resourceType]
			if !ok {
				mgmtPerms = defaultManagementPermissions
			}

			if len(mgmtPerms) == 0 {
				// don't have resources like this, but if we need to disable this mutator for a given resource:
				return v, nil
			}

			// Process permissions array
			updatedPermissions, err := processPermissions(permissions, currentUser, mgmtPerms)
			if err != nil {
				return dyn.InvalidValue, err
			}

			// Set the updated permissions back
			return dyn.Set(v, "permissions", updatedPermissions)
		})
	}
}

func readUser(v dyn.Value) string {
	userName, _ := dyn.Get1(v, "user_name").AsString()
	if userName != "" {
		return userName
	}
	servicePrincipalName, _ := dyn.Get1(v, "service_principal_name").AsString()
	return servicePrincipalName
}

func processPermissions(permissions dyn.Value, currentUser string, mgmtPerms []string) (dyn.Value, error) {
	hasIsOwner := false
	permissionArray := permissions.MustSequence()
	for _, permission := range permissionArray {
		level, ok := dyn.Get1(permission, "level").AsString()
		if !ok {
			continue
		}
		if level == isOwner {
			hasIsOwner = true
		}
		user := readUser(permission)
		if user == "" || user != currentUser {
			continue
		}
		if slices.Contains(mgmtPerms, level) {
			return permissions, nil
		}
	}

	permissionToAdd := mgmtPerms[0]
	if hasIsOwner && permissionToAdd == isOwner {
		if len(mgmtPerms) > 1 {
			permissionToAdd = mgmtPerms[1]
		} else {
			// do not add second IS_OWNER, keep existing IS_OWNER
			permissionToAdd = ""
		}
	}

	if permissionToAdd == "" {
		return permissions, nil
	}

	permissionArray = append(permissionArray, createPermission(currentUser, permissionToAdd))
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
		rv, err := dyn.Get(v, "resources")
		if err != nil {
			// If the resources key is not found, we can skip this mutator.
			if dyn.IsNoSuchKeyError(err) {
				return v, nil
			}

			return dyn.InvalidValue, err
		}

		// Walk the resources and ensure current user has correct permissions
		nv, err := dyn.Walk(rv, ensureCurrentUserPermission(currentUser))
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Set the resources with the updated permissions back into the bundle
		return dyn.Set(v, "resources", nv)
	})

	return diag.FromErr(err)
}
