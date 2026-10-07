package resourcemutator

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	err := structvar.ForEach(b.Config.View(), structpath.MustParsePattern("resources.*.*.permissions"), func(p *structpath.PathNode, v structvar.View) error {
		// Extract resource type from path: resources.<resource_type>.<resource_name>.permissions
		resourceType := p.KeyAt(1)
		if ignoredResources[resourceType] || v.Kind() != structvar.KindSequence {
			return nil
		}

		var permissions []permission
		for _, pv := range v.Sequence() {
			var perm permission
			perm.level, _ = pv.Get("level").AsString()
			perm.userName, _ = pv.Get("user_name").AsString()
			perm.servicePrincipalName, _ = pv.Get("service_principal_name").AsString()
			perm.groupName, _ = pv.Get("group_name").AsString()
			permissions = append(permissions, perm)
		}

		err := setPermissions(v, processPermissions(currentUser, resourceType, permissions))
		if err != nil {
			return err
		}

		// The permissions are rebuilt without locations.
		b.Config.SetLocations(p, nil)
		return nil
	})

	return diag.FromErr(err)
}

// setPermissions replaces the permissions described by v. The type of the permissions
// differs between resources, but all have the same fields.
func setPermissions(v structvar.View, permissions []permission) error {
	field := v.Reflect()
	for field.Kind() == reflect.Pointer || field.Kind() == reflect.Interface {
		field = field.Elem()
	}
	if !field.CanSet() || field.Kind() != reflect.Slice {
		return fmt.Errorf("cannot set permissions of type %s", field.Type())
	}

	out := reflect.MakeSlice(field.Type(), len(permissions), len(permissions))
	for i, p := range permissions {
		elem := out.Index(i)
		elem.FieldByName("Level").SetString(p.level)
		elem.FieldByName("UserName").SetString(p.userName)
		elem.FieldByName("ServicePrincipalName").SetString(p.servicePrincipalName)
		elem.FieldByName("GroupName").SetString(p.groupName)
	}
	field.Set(out)
	return nil
}
