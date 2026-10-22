package dresources

import (
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
)

var SupportedResources = map[string]any{
	"jobs":                        (*ResourceJob)(nil),
	"job_runs":                    (*ResourceJobRun)(nil),
	"pipelines":                   (*ResourcePipeline)(nil),
	"experiments":                 (*ResourceExperiment)(nil),
	"features":                    (*ResourceFeature)(nil),
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
	"secret_scopes.permissions": (*ResourceSecretScopeAcls)(nil),

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

func init() {
	for name, description := range resources.All {
		if p := description.Permissions; p.RequestObjectType != "" {
			SupportedResources[name+".permissions"] = (*ResourcePermissions)(nil)
		}
	}
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
