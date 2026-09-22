package dresources

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/calladapt"
	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/databricks/databricks-sdk-go"
)

var SupportedResources = map[string]any{
	"jobs":                        (*ResourceJob)(nil),
	"job_runs":                    (*ResourceJobRun)(nil),
	"pipelines":                   (*ResourcePipeline)(nil),
	"experiments":                 (*ResourceExperiment)(nil),
	"catalogs":                    (*ResourceCatalog)(nil),
	"schemas":                     (*ResourceSchema)(nil),
	"external_locations":          (*ResourceExternalLocation)(nil),
	"volumes":                     (*ResourceVolume)(nil),
	"models":                      (*ResourceMlflowModel)(nil),
	"apps":                        (*ResourceApp)(nil),
	"sql_warehouses":              (*ResourceSqlWarehouse)(nil),
	"database_instances":          (*ResourceDatabaseInstance)(nil),
	"database_catalogs":           (*ResourceDatabaseCatalog)(nil),
	"synced_database_tables":      (*ResourceSyncedDatabaseTable)(nil),
	"postgres_projects":           (*ResourcePostgresProject)(nil),
	"postgres_branches":           (*ResourcePostgresBranch)(nil),
	"postgres_endpoints":          (*ResourcePostgresEndpoint)(nil),
	"postgres_catalogs":           (*ResourcePostgresCatalog)(nil),
	"postgres_databases":          (*ResourcePostgresDatabase)(nil),
	"postgres_roles":              (*ResourcePostgresRole)(nil),
	"postgres_synced_tables":      (*ResourcePostgresSyncedTable)(nil),
	"postgres_snapshot_schedules": (*ResourcePostgresSnapshotSchedule)(nil),
	"alerts":                      (*ResourceAlert)(nil),
	"clusters":                    (*ResourceCluster)(nil),
	"registered_models":           (*ResourceRegisteredModel)(nil),
	"dashboards":                  (*ResourceDashboard)(nil),
	"genie_spaces":                (*ResourceGenieSpace)(nil),
	"secret_scopes":               (*ResourceSecretScope)(nil),
	"model_serving_endpoints":     (*ResourceModelServingEndpoint)(nil),
	"model_services":              (*ResourceModelService)(nil),
	"mcp_services":                (*ResourceMcpService)(nil),
	"model_provider_services":     (*ResourceModelProviderService)(nil),
	"quality_monitors":            (*ResourceQualityMonitor)(nil),
	"vector_search_endpoints":     (*ResourceVectorSearchEndpoint)(nil),
	"vector_search_indexes":       (*ResourceVectorSearchIndex)(nil),
	"instance_pools":              (*ResourceInstancePool)(nil),
	"secrets":                     (*ResourceSecret)(nil),
	"cluster_policies":            (*ResourceClusterPolicy)(nil),

	// Permissions
	"jobs.permissions":                    (*ResourcePermissions)(nil),
	"pipelines.permissions":               (*ResourcePermissions)(nil),
	"apps.permissions":                    (*ResourcePermissions)(nil),
	"alerts.permissions":                  (*ResourcePermissions)(nil),
	"clusters.permissions":                (*ResourcePermissions)(nil),
	"cluster_policies.permissions":        (*ResourcePermissions)(nil),
	"database_instances.permissions":      (*ResourcePermissions)(nil),
	"postgres_projects.permissions":       (*ResourcePermissions)(nil),
	"experiments.permissions":             (*ResourcePermissions)(nil),
	"models.permissions":                  (*ResourcePermissions)(nil),
	"sql_warehouses.permissions":          (*ResourcePermissions)(nil),
	"secret_scopes.permissions":           (*ResourceSecretScopeAcls)(nil),
	"model_serving_endpoints.permissions": (*ResourcePermissions)(nil),
	"dashboards.permissions":              (*ResourcePermissions)(nil),
	"genie_spaces.permissions":            (*ResourcePermissions)(nil),
	"vector_search_endpoints.permissions": (*ResourcePermissions)(nil),
	"instance_pools.permissions":          (*ResourcePermissions)(nil),

	// Grants
	"catalogs.grants":                (*ResourceGrants)(nil),
	"schemas.grants":                 (*ResourceGrants)(nil),
	"external_locations.grants":      (*ResourceGrants)(nil),
	"volumes.grants":                 (*ResourceGrants)(nil),
	"registered_models.grants":       (*ResourceGrants)(nil),
	"vector_search_indexes.grants":   (*ResourceGrants)(nil),
	"secrets.grants":                 (*ResourceGrants)(nil),
	"model_services.grants":          (*ResourceGrants)(nil),
	"mcp_services.grants":            (*ResourceGrants)(nil),
	"model_provider_services.grants": (*ResourceGrants)(nil),

	// Internal resources
	"internal_immutable_snapshots": (*ResourceSnapshot)(nil),
}

// copiers holds an auto-generated RemapState copier for every resource whose
// remote type differs from its state type and that does not supply a custom RemapState.
// It is built once, at package initialization, from SupportedResources — so a resource
// whose types cannot be safely copied fails at load, not at deploy time, and no separate
// test can forget to cover it. Resources that need real remapping logic keep a custom
// RemapState method and are skipped here.
var copiers = buildCopiers()

func buildCopiers() map[reflect.Type]*structcopy.Copier {
	iface := reflect.TypeFor[IResource]()
	out := make(map[reflect.Type]*structcopy.Copier)
	var errs []string

	for resourceType, resource := range SupportedResources {
		implType := reflect.TypeOf(resource)
		if _, done := out[implType]; done {
			continue // same implementation registered under several keys (permissions, grants)
		}

		remap, err := calladapt.PrepareCall(resource, iface, "RemapState")
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: RemapState: %v", resourceType, err))
			continue
		}
		if remap != nil {
			continue // custom override
		}

		prepareState, err := calladapt.PrepareCall(resource, iface, "PrepareState")
		if err != nil || prepareState == nil {
			errs = append(errs, fmt.Sprintf("%s: PrepareState: %v", resourceType, err))
			continue
		}
		doRead, err := calladapt.PrepareCall(resource, iface, "DoRead")
		if err != nil || doRead == nil {
			errs = append(errs, fmt.Sprintf("%s: DoRead: %v", resourceType, err))
			continue
		}

		stateType := prepareState.OutTypes[0]
		remoteType := doRead.OutTypes[0]
		if remoteType == stateType {
			continue // identity: the adapter returns the remote unchanged, no copier needed
		}

		copier, err := structcopy.Compile(remoteType, stateType)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v (implement RemapState for this resource)", resourceType, err))
			continue
		}
		out[implType] = copier
	}

	if len(errs) > 0 {
		slices.Sort(errs)
		panic("dresources: cannot build RemapState copiers:\n" + strings.Join(errs, "\n"))
	}
	return out
}

func InitAll(client *databricks.WorkspaceClient) (map[string]*Adapter, error) {
	result := make(map[string]*Adapter)
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, client)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", resourceType, err)
		}
		result[resourceType] = adapter
	}
	return result, nil
}
