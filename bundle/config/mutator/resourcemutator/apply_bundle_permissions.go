package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
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

	keys := make([]string, 0, len(levelsMap))
	for key := range levelsMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		if err := applyBundlePermissionsTo(ctx, b, key); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// applyBundlePermissionsTo applies the bundle permissions to the resources of the
// given type.
func applyBundlePermissionsTo(ctx context.Context, b *bundle.Bundle, key string) error {
	r := &b.Config.Resources
	switch key {
	case "jobs":
		return applyToAll(ctx, b, key, r.Jobs, func(x *resources.Job) *[]resources.JobPermission { return &x.Permissions })
	case "pipelines":
		return applyToAll(ctx, b, key, r.Pipelines, func(x *resources.Pipeline) *[]resources.PipelinePermission { return &x.Permissions })
	case "experiments":
		return applyToAll(ctx, b, key, r.Experiments, func(x *resources.MlflowExperiment) *[]resources.MlflowExperimentPermission { return &x.Permissions })
	case "models":
		return applyToAll(ctx, b, key, r.Models, func(x *resources.MlflowModel) *[]resources.MlflowModelPermission { return &x.Permissions })
	case "model_serving_endpoints":
		return applyToAll(ctx, b, key, r.ModelServingEndpoints, func(x *resources.ModelServingEndpoint) *[]resources.ModelServingEndpointPermission {
			return &x.Permissions
		})
	case "dashboards":
		return applyToAll(ctx, b, key, r.Dashboards, func(x *resources.Dashboard) *[]resources.Permission { return &x.Permissions })
	case "genie_spaces":
		return applyToAll(ctx, b, key, r.GenieSpaces, func(x *resources.GenieSpace) *[]resources.Permission { return &x.Permissions })
	case "apps":
		return applyToAll(ctx, b, key, r.Apps, func(x *resources.App) *[]resources.AppPermission { return &x.Permissions })
	case "secret_scopes":
		return applyToAll(ctx, b, key, r.SecretScopes, func(x *resources.SecretScope) *[]resources.SecretScopePermission { return &x.Permissions })
	case "alerts":
		return applyToAll(ctx, b, key, r.Alerts, func(x *resources.Alert) *[]resources.Permission { return &x.Permissions })
	case "sql_warehouses":
		return applyToAll(ctx, b, key, r.SqlWarehouses, func(x *resources.SqlWarehouse) *[]resources.SqlWarehousePermission { return &x.Permissions })
	case "database_instances":
		return applyToAll(ctx, b, key, r.DatabaseInstances, func(x *resources.DatabaseInstance) *[]resources.Permission { return &x.Permissions })
	case "postgres_projects":
		return applyToAll(ctx, b, key, r.PostgresProjects, func(x *resources.PostgresProject) *[]resources.Permission { return &x.Permissions })
	case "clusters":
		return applyToAll(ctx, b, key, r.Clusters, func(x *resources.Cluster) *[]resources.ClusterPermission { return &x.Permissions })
	case "vector_search_endpoints":
		return applyToAll(ctx, b, key, r.VectorSearchEndpoints, func(x *resources.VectorSearchEndpoint) *[]resources.VectorSearchEndpointPermission {
			return &x.Permissions
		})
	case "instance_pools":
		return applyToAll(ctx, b, key, r.InstancePools, func(x *resources.InstancePool) *[]resources.InstancePoolPermission { return &x.Permissions })
	default:
		return fmt.Errorf("bundle permissions are not supported for %s", key)
	}
}

// applyToAll applies the bundle permissions to the resources in m, in key order.
func applyToAll[R any, P ~resources.PermissionFields[L], L ~string](ctx context.Context, b *bundle.Bundle, key string, m map[string]*R, perms func(*R) *[]P) error {
	for _, name := range slices.Sorted(maps.Keys(m)) {
		if x := m[name]; x != nil {
			if err := applyToResource(ctx, b, key, name, perms(x)); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyToResource applies the bundle permissions to the permissions perms of the
// resource name of type key.
func applyToResource[P ~resources.PermissionFields[L], L ~string](ctx context.Context, b *bundle.Bundle, key, name string, perms *[]P) error {
	path := structpath.NewPath(nil, "resources", key, name, "permissions")
	// Present as [structvar.View] decides: a non-nil list or a pure reference.
	had := *perms != nil || b.Config.IsReferenceAt(path)

	existing := make([]resources.Permission, 0, len(*perms))
	for _, p := range *perms {
		f := resources.PermissionFields[L](p)
		existing = append(existing, resources.Permission{
			Level:                iam.PermissionLevel(f.Level),
			UserName:             f.UserName,
			GroupName:            f.GroupName,
			ServicePrincipalName: f.ServicePrincipalName,
		})
	}

	added := convertPermissions(ctx, b.Config.Permissions, existing, key, levelsMap[key])
	for _, p := range added {
		*perms = append(*perms, P(resources.PermissionFields[L]{
			Level:                L(p.Level),
			UserName:             p.UserName,
			GroupName:            p.GroupName,
			ServicePrincipalName: p.ServicePrincipalName,
		}))
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
