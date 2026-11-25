package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

var allowedLevels = []string{permissions.CAN_MANAGE, permissions.CAN_VIEW, permissions.CAN_RUN}

// bundleLevels returns the levels the bundle-level permissions map to for the resource
// type key, or false if the type accepts no bundle-level permissions.
func bundleLevels(key string) (resources.BundlePermissionLevels, bool) {
	levels := permissionsSupport(key).BundleLevels
	return levels, levels != resources.BundlePermissionLevels{}
}

// resourceLevel returns the level of a resource type that the bundle-level permission
// level maps to, or an empty string if it does not apply to the type.
func resourceLevel(levels resources.BundlePermissionLevels, bundleLevel string) string {
	switch bundleLevel {
	case permissions.CAN_MANAGE:
		return levels.CanManage
	case permissions.CAN_VIEW:
		return levels.CanView
	case permissions.CAN_RUN:
		return levels.CanRun
	}
	return ""
}

type bundlePermissions struct{}

func ApplyBundlePermissions() bundle.Mutator {
	return &bundlePermissions{}
}

func (m *bundlePermissions) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := validatePermissions(b)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, group := range b.Config.Resources.AllResources() {
		levels, ok := bundleLevels(group.Description.PluralName)
		if !ok {
			continue
		}

		for _, name := range slices.Sorted(maps.Keys(group.Resources)) {
			field := reflect.ValueOf(group.Resources[name]).Elem().FieldByName("Permissions")
			permissions := readPermissions(field)
			permissions = append(permissions, convertPermissions(
				b.Config.Permissions,
				permissions,
				levels,
			)...)
			writePermissions(field, permissions)
		}
	}

	return nil
}

// readPermissions converts the permissions field of a resource to []resources.Permission.
// The element type of the field differs between resources, but all have the same fields.
func readPermissions(field reflect.Value) []resources.Permission {
	var permissions []resources.Permission
	for i := range field.Len() {
		p := field.Index(i)
		permissions = append(permissions, resources.Permission{
			Level:                iam.PermissionLevel(p.FieldByName("Level").String()),
			UserName:             p.FieldByName("UserName").String(),
			ServicePrincipalName: p.FieldByName("ServicePrincipalName").String(),
			GroupName:            p.FieldByName("GroupName").String(),
		})
	}
	return permissions
}

// writePermissions sets the permissions field of a resource to permissions.
// An empty list is stored as nil, so that the field is absent from the configuration.
func writePermissions(field reflect.Value, permissions []resources.Permission) {
	if len(permissions) == 0 {
		field.SetZero()
		return
	}

	out := reflect.MakeSlice(field.Type(), len(permissions), len(permissions))
	for i, p := range permissions {
		e := out.Index(i)
		e.FieldByName("Level").SetString(string(p.Level))
		e.FieldByName("UserName").SetString(p.UserName)
		e.FieldByName("ServicePrincipalName").SetString(p.ServicePrincipalName)
		e.FieldByName("GroupName").SetString(p.GroupName)
	}
	field.Set(out)
}

func validatePermissions(b *bundle.Bundle) error {
	for _, p := range b.Config.Permissions {
		if !slices.Contains(allowedLevels, string(p.Level)) {
			return fmt.Errorf("invalid permission level: %s, allowed values: [%s]", p.Level, strings.Join(allowedLevels, ", "))
		}
	}

	return nil
}

func (m *bundlePermissions) Name() string {
	return "ApplyBundlePermissions"
}

func convertPermissions(
	bundlePermissions []resources.Permission,
	resourcePermissions []resources.Permission,
	levels resources.BundlePermissionLevels,
) []resources.Permission {
	var permissions []resources.Permission
	for _, p := range bundlePermissions {
		level := resourceLevel(levels, string(p.Level))
		// If there is no bundle permission level defined for the resource, it means
		// it's not applicable for the resource, therefore skipping
		if level == "" {
			continue
		}

		if isPermissionOverlap(p, resourcePermissions) {
			continue
		}

		permissions = append(permissions, resources.Permission{
			Level:                iam.PermissionLevel(level),
			UserName:             p.UserName,
			GroupName:            p.GroupName,
			ServicePrincipalName: p.ServicePrincipalName,
		})
	}

	return permissions
}

func isPermissionOverlap(
	permission resources.Permission,
	resourcePermissions []resources.Permission,
) bool {
	for _, rp := range resourcePermissions {
		if rp.GroupName != "" && rp.GroupName == permission.GroupName {
			return true
		}

		if rp.UserName != "" && rp.UserName == permission.UserName {
			return true
		}

		if rp.ServicePrincipalName != "" && rp.ServicePrincipalName == permission.ServicePrincipalName {
			return true
		}
	}

	return false
}
