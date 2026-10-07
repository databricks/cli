package resourcemutator

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

var (
	allowedLevels = []string{permissions.CAN_MANAGE, permissions.CAN_VIEW, permissions.CAN_RUN}
	// Map of allowed permission levels to the corresponding permission level of specific resources
	levelsMap = map[string](map[string]string){
		"jobs": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_VIEW",
			permissions.CAN_RUN:    "CAN_MANAGE_RUN",
		},
		"pipelines": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_VIEW",
			permissions.CAN_RUN:    "CAN_RUN",
		},
		"experiments": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_READ",
		},
		"models": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_READ",
		},
		"model_serving_endpoints": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_VIEW",
			permissions.CAN_RUN:    "CAN_QUERY",
		},
		"dashboards": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_READ",
		},
		"genie_spaces": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_READ",
		},
		"apps": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_USE",
		},
		"secret_scopes": {
			permissions.CAN_MANAGE: "MANAGE",
			permissions.CAN_VIEW:   "READ",
		},
		"alerts": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_READ",
			permissions.CAN_RUN:    "CAN_RUN",
		},
		"sql_warehouses": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_VIEW",
			permissions.CAN_RUN:    "CAN_MONITOR",
		},
		"database_instances": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_USE",
		},
		"postgres_projects": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_USE",
		},
		"clusters": {
			// https://docs.databricks.com/aws/en/security/auth/access-control/#compute-acls
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_ATTACH_TO",
			permissions.CAN_RUN:    "CAN_RESTART",
		},
		"vector_search_endpoints": {
			// https://docs.databricks.com/aws/en/security/auth/access-control/#vector-search-endpoint-acls
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_USE",
		},
		"instance_pools": {
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_ATTACH_TO",
		},
	}
)

type bundlePermissions struct{}

func ApplyBundlePermissions() bundle.Mutator {
	return &bundlePermissions{}
}

func (m *bundlePermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := validatePermissions(b)
	if err != nil {
		return diag.FromErr(err)
	}

	keys := make([]string, 0, len(levelsMap))
	for key := range levelsMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		pattern := structpath.MustParsePattern("resources." + key + ".*")

		err = structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, v structvar.View) error {
			had := v.Get("permissions").IsValid()
			var permissions []resources.Permission
			for _, pv := range v.Get("permissions").Sequence() {
				level, _ := pv.Get("level").AsString()
				userName, _ := pv.Get("user_name").AsString()
				groupName, _ := pv.Get("group_name").AsString()
				servicePrincipalName, _ := pv.Get("service_principal_name").AsString()
				permissions = append(permissions, resources.Permission{
					Level:                iam.PermissionLevel(level),
					UserName:             userName,
					GroupName:            groupName,
					ServicePrincipalName: servicePrincipalName,
				})
			}

			added := convertPermissions(
				ctx,
				b.Config.Permissions,
				permissions,
				key,
				levelsMap[key],
			)
			if len(added) > 0 {
				if err := appendPermissions(v, added); err != nil {
					return err
				}
			}

			if !had {
				return nil
			}

			// Empty permissions are dropped. Otherwise they are rebuilt without locations.
			permissionsPath := structpath.NewStringKey(p, "permissions")
			if len(permissions) == 0 && len(added) == 0 {
				return b.Config.Delete(permissionsPath)
			}
			b.Config.SetLocations(permissionsPath, nil)
			return nil
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// appendPermissions appends permissions to the permissions field of the resource described by v.
// The type of the permissions differs between resources, but all have the same fields.
func appendPermissions(v structvar.View, permissions []resources.Permission) error {
	r := v.Reflect()
	for r.Kind() == reflect.Pointer || r.Kind() == reflect.Interface {
		r = r.Elem()
	}
	field := r.FieldByName("Permissions")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Slice {
		return fmt.Errorf("cannot set permissions of %s", r.Type())
	}

	for _, p := range permissions {
		elem := reflect.New(field.Type().Elem()).Elem()
		elem.FieldByName("Level").SetString(string(p.Level))
		elem.FieldByName("UserName").SetString(p.UserName)
		elem.FieldByName("GroupName").SetString(p.GroupName)
		elem.FieldByName("ServicePrincipalName").SetString(p.ServicePrincipalName)
		field.Set(reflect.Append(field, elem))
	}
	return nil
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
	ctx context.Context,
	bundlePermissions []resources.Permission,
	resourcePermissions []resources.Permission,
	resourceName string,
	lm map[string]string,
) []resources.Permission {
	var permissions []resources.Permission
	for _, p := range bundlePermissions {
		level, ok := lm[string(p.Level)]
		// If there is no bundle permission level defined in the map, it means
		// it's not applicable for the resource, therefore skipping
		if !ok {
			continue
		}

		if notifyForPermissionOverlap(ctx, p, resourcePermissions, resourceName) {
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
	resourceName string,
) (bool, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	for _, rp := range resourcePermissions {
		if rp.GroupName != "" && rp.GroupName == permission.GroupName {
			diagnostics = diagnostics.Extend(
				diag.Warningf("'%s' already has permissions set for '%s' group", resourceName, rp.GroupName),
			)
		}

		if rp.UserName != "" && rp.UserName == permission.UserName {
			diagnostics = diagnostics.Extend(
				diag.Warningf("'%s' already has permissions set for '%s' user name", resourceName, rp.UserName),
			)
		}

		if rp.ServicePrincipalName != "" && rp.ServicePrincipalName == permission.ServicePrincipalName {
			diagnostics = diagnostics.Extend(
				diag.Warningf("'%s' already has permissions set for '%s' service principal name", resourceName, rp.ServicePrincipalName),
			)
		}
	}

	return len(diagnostics) > 0, diagnostics
}

func notifyForPermissionOverlap(
	ctx context.Context,
	permission resources.Permission,
	resourcePermissions []resources.Permission,
	resourceName string,
) bool {
	isOverlap, diagnostics := isPermissionOverlap(permission, resourcePermissions, resourceName)
	for _, d := range diagnostics {
		logdiag.LogDiag(ctx, d)
	}

	return isOverlap
}
