package resourcemutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/iam"
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
	// Cluster policies only support CAN_USE; injecting the current user as
	// CAN_MANAGE/IS_OWNER would be rejected by the permissions API.
	"cluster_policies": true,
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
// 5. Any given principal should have at most one permission level set.

type fixPermissions struct{}

// This mutator ensures the current user has the correct permissions for deployed resources.
func FixPermissions() bundle.Mutator {
	return &fixPermissions{}
}

func (m *fixPermissions) Name() string {
	return "FixPermissions"
}

// permission is a permission entry: a level and the principal it is granted to.
type permission struct {
	level                string
	userName             string
	servicePrincipalName string
	groupName            string
}

func (p permission) user() string {
	if p.userName != "" {
		return p.userName
	}
	return p.servicePrincipalName
}

// principal returns the principal in the form "<field>:<name>", or "" if there is none.
func (p permission) principal() string {
	switch {
	case p.userName != "":
		return "user_name:" + p.userName
	case p.servicePrincipalName != "":
		return "service_principal_name:" + p.servicePrincipalName
	case p.groupName != "":
		return "group_name:" + p.groupName
	}
	return ""
}

func processPermissions(currentUser, resourceType string, permissions []permission) []permission {
	return useMaximumLevel(ensureCurrentUserMgmtPermissions(permissions, currentUser, resourceType))
}

func ensureCurrentUserMgmtPermissions(permissions []permission, currentUser, resourceType string) []permission {
	currentUserHasIsOwner := false
	currentUserIndCanManage := -1
	canAddIsOwner := hasIsOwner[resourceType]

	for ind, p := range permissions {
		if p.level == "" {
			continue
		}
		user := p.user()
		if p.level == isOwner {
			canAddIsOwner = false
			if user == currentUser {
				currentUserHasIsOwner = true
			}
		}
		if user == currentUser && p.level == canManage {
			currentUserIndCanManage = ind
		}
	}

	if currentUserHasIsOwner {
		return permissions
	}

	if canAddIsOwner {
		if currentUserIndCanManage >= 0 {
			// Upgrade current user's CAN_MANAGE to IS_OWNER. We do this because terraform will add IS_OWNER if it does not see one
			// and that may confuse backend. We can stop doing it when removed terraform.
			permissions[currentUserIndCanManage].level = isOwner
		} else {
			permissions = append(permissions, createPermission(currentUser, isOwner))
		}
		return permissions
	}

	if currentUserIndCanManage < 0 {
		permissions = append(permissions, createPermission(currentUser, canManage))
	}

	return permissions
}

func useMaximumLevel(permissions []permission) []permission {
	levelPerPrincipal := make(map[string]string)
	seen := make(map[string]bool)
	var principals []string

	for _, p := range permissions {
		if p.level == "" {
			continue
		}

		principal := p.principal()
		if principal == "" {
			continue
		}
		if !seen[principal] {
			seen[principal] = true
			principals = append(principals, principal)
		}
		levelPerPrincipal[principal] = resources.GetMaxLevel(levelPerPrincipal[principal], p.level)
	}

	var newPermissions []permission
	for _, principal := range principals {
		newPermissions = append(newPermissions, createPermissionFromPrincipal(principal, levelPerPrincipal[principal]))
	}

	return newPermissions
}

func createPermission(user, level string) permission {
	// Determine if currentUser is a service principal or user
	if iamutil.IsServicePrincipalName(user) {
		return permission{level: level, servicePrincipalName: user}
	}
	return permission{level: level, userName: user}
}

func createPermissionFromPrincipal(principal, level string) permission {
	items := strings.SplitN(principal, ":", 2)
	field := items[0]
	value := items[1]
	p := permission{level: level}
	switch field {
	case "user_name":
		p.userName = value
	case "service_principal_name":
		p.servicePrincipalName = value
	case "group_name":
		p.groupName = value
	}
	return p
}

func (m *fixPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// CurrentUser is populated by PopulateCurrentUser early in the initialize phase.
	// It can be nil when this mutator runs outside that phase (e.g. NormalizeResources
	// after PythonMutator); there is no user to add as owner, so skip.
	if b.Config.Workspace.CurrentUser == nil {
		return nil
	}
	currentUser := b.Config.Workspace.CurrentUser.UserName

	b.Config.Resources.IterateAllPermissions(func(typ, name string, perms config.PermissionList) bool {
		if ignoredResources[typ] || perms.IsNil() {
			return true
		}

		permissions := make([]permission, 0, perms.Len())
		for i := range perms.Len() {
			p := perms.At(i)
			permissions = append(permissions, permission{
				level:                string(p.Level),
				userName:             p.UserName,
				servicePrincipalName: p.ServicePrincipalName,
				groupName:            p.GroupName,
			})
		}

		processed := processPermissions(currentUser, typ, permissions)
		out := make([]resources.Permission, 0, len(processed))
		for _, p := range processed {
			out = append(out, resources.Permission{
				Level:                iam.PermissionLevel(p.level),
				UserName:             p.userName,
				ServicePrincipalName: p.servicePrincipalName,
				GroupName:            p.groupName,
			})
		}
		perms.Set(out)

		// The permissions are rebuilt without locations.
		b.Config.SetLocations(structpath.NewPath(nil, "resources", typ, name, "permissions"), nil)
		return true
	})

	return nil
}
