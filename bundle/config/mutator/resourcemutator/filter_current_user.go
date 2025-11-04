package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/iamutil"
)

const (
	isOwner   = "IS_OWNER"
	canManage = "CAN_MANAGE"
)

// Which resources support IS_OWNER permission.
var hasIsOwner = map[string]bool{
	"jobs":           true,
	"pipelines":      true,
	"sql_warehouses": true,
}

var ignoredResources = map[string]bool{
	"secret_scopes": true,
}

// When processing permissions, we need to implement these constraints:
// 1. There should be no more than one IS_OWNER settings for a given resource.
// 2. We should automatically add IS_OWNER for current user (for resources that support it) or CAN_MANAGE permission.
//    Terraform will do this, so doing this early makes request equal between terraform and direct.
// 3. We prefer to add IS_OWNER, unless there already is one, in which case we fallback to CAN_MANAGE.
// 4. Current user cannot have both CAN_MANAGE and IS_OWNER, we've seen backend failing with
//    "Error: cannot create permissions: Permissions being set for UserName([USERNAME]) are ambiguous"
//    Since terraform adds IS_OWNER permission when there is not one, regardless of CAN_MANAGE presence,
//    he above error can occur. We thus add another bit of logic: we upgrade CAN_MANAGE to IS_OWNER when we can.

type ensureOwnerPermissions struct{}

// This mutator ensures the current user has the correct permissions for deployed resources.
func EnsureOwnerPermissions() bundle.Mutator {
	return &ensureOwnerPermissions{}
}

func (m *ensureOwnerPermissions) Name() string {
	return "EnsureOwnerPermissions"
}

func ensureCurrentUserPermission(currentUser string) dyn.MapFunc {
	return func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Extract resource type from path: resources.<resource_type>.<resource_name>.permissions
		if len(p) != 4 || p[0].Key() != "resources" || p[3].Key() != "permissions" {
			return v, nil
		}

		resourceType := p[1].Key()
		if ignoredResources[resourceType] {
			return v, nil
		}

		return processPermissions(v, currentUser, resourceType)
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

func processPermissions(permissions dyn.Value, currentUser, resourceType string) (dyn.Value, error) {
	currentUserHasIsOwner := false
	currentUserIndCanManage := -1
	canAddIsOwner := hasIsOwner[resourceType]

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
		if level == isOwner {
			canAddIsOwner = false
			if user == currentUser {
				currentUserHasIsOwner = true
			}
		}
		if user == currentUser && level == canManage {
			currentUserIndCanManage = ind
		}
	}

	if currentUserHasIsOwner {
		return dyn.V(permissionArray), nil
	}

	if canAddIsOwner {
		if currentUserIndCanManage >= 0 {
			// Upgrade current user's CAN_MANAGE to IS_OWNER. We do this because terraform will add IS_OWNER if it does not see one
			// and that may confuse backend. We can stop doing it when removed terraform.
			v, _ := dyn.Set(permissionArray[currentUserIndCanManage], "level", dyn.V(isOwner))
			permissionArray[currentUserIndCanManage] = v
		} else {
			permissionArray = append(permissionArray, createPermission(currentUser, isOwner))
		}
		return dyn.V(permissionArray), nil
	}

	if currentUserIndCanManage < 0 {
		permissionArray = append(permissionArray, createPermission(currentUser, canManage))
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

func (m *ensureOwnerPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
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
