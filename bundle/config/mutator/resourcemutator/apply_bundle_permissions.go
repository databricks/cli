package resourcemutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
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

	b.Config.Resources.IterateAllPermissions(func(typ, name string, perms config.PermissionList) bool {
		lm, ok := levelsMap[typ]
		if !ok {
			return true
		}
		err = applyToResource(ctx, b, typ, name, lm, perms)
		return err == nil
	})
	return diag.FromErr(err)
}

// applyToResource applies the bundle permissions to the permissions perms of the
// resource name of type typ, using the permission levels lm of that type.
func applyToResource(ctx context.Context, b *bundle.Bundle, typ, name string, lm map[string]string, perms config.PermissionList) error {
	path := structpath.NewPath(nil, "resources", typ, name, "permissions")
	// Present as [structvar.View] decides: a non-nil list or a pure reference.
	had := !perms.IsNil() || b.Config.IsReferenceAt(path)

	existing := make([]resources.Permission, 0, perms.Len())
	for i := range perms.Len() {
		existing = append(existing, perms.At(i))
	}

	added := convertPermissions(ctx, b.Config.Permissions, existing, typ, lm)
	for _, p := range added {
		perms.Append(p)
	}

	if !had {
		return nil
	}

	// Empty permissions are dropped. Otherwise they are rebuilt without locations.
	if len(existing) == 0 && len(added) == 0 {
		return b.Config.Delete(path)
	}
	b.Config.SetLocations(path, nil)
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
