package resourcemutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

var (
	allowedLevels = []string{permissions.CAN_MANAGE, permissions.CAN_VIEW, permissions.CAN_RUN}
	levelsMap     = map[string](map[string]string){
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
			// Genie spaces also support CAN_EDIT, but bundle-level permissions
			// don't have an equivalent. Users can set CAN_EDIT directly on the resource.
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_VIEW",
			permissions.CAN_RUN:    "CAN_RUN",
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
		"clusters": {
			// https://docs.databricks.com/aws/en/security/auth/access-control/#compute-acls
			permissions.CAN_MANAGE: "CAN_MANAGE",
			permissions.CAN_VIEW:   "CAN_ATTACH_TO",
			permissions.CAN_RUN:    "CAN_RESTART",
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

	patterns := make(map[string]dyn.Pattern, 0)
	for key := range levelsMap {
		patterns[key] = dyn.NewPattern(
			dyn.Key("resources"),
			dyn.Key(key),
			dyn.AnyKey(),
		)
	}

	err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		for key, pattern := range patterns {
			v, err = dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				var permissions []resources.Permission
				pv, err := dyn.Get(v, "permissions")
				// If the permissions field is not found, we set to an empty array
				if err != nil {
					pv = dyn.V([]dyn.Value{})
				}

				err = convert.ToTyped(&permissions, pv)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to convert permissions: %w", err)
				}

				permissions = append(permissions, convertPermissions(
					ctx,
					b.Config.Permissions,
					permissions,
					key,
					levelsMap[key],
				)...)

				if len(permissions) == 0 {
					permissions = nil
				}

				pv, err = convert.FromTyped(permissions, dyn.NilValue)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to convert permissions: %w", err)
				}

				return dyn.Set(v, "permissions", pv)
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func validatePermissions(b *bundle.Bundle) error {
	for _, p := range b.Config.Permissions {
		if !slices.Contains(allowedLevels, p.Level) {
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
		level, ok := lm[p.Level]
		// If there is no bundle permission level defined in the map, it means
		// it's not applicable for the resource, therefore skipping
		if !ok {
			continue
		}

		if notifyForPermissionOverlap(ctx, p, resourcePermissions, resourceName) {
			continue
		}

		permissions = append(permissions, resources.Permission{
			Level:                level,
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
	isOverlap, _ := isPermissionOverlap(permission, resourcePermissions, resourceName)
	// TODO: When we start to collect all diagnostics at the top level and visualize jointly,
	// use diagnostics returned from isPermissionOverlap to display warnings

	return isOverlap
}
