package dresources

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

var SupportedResources = map[string]any{
	"jobs":                   (*ResourceJob)(nil),
	"pipelines":              (*ResourcePipeline)(nil),
	"experiments":            (*ResourceExperiment)(nil),
	"schemas":                (*ResourceSchema)(nil),
	"volumes":                (*ResourceVolume)(nil),
	"models":                 (*ResourceMlflowModel)(nil),
	"apps":                   (*ResourceApp)(nil),
	"sql_warehouses":         (*ResourceSqlWarehouse)(nil),
	"database_instances":     (*ResourceDatabaseInstance)(nil),
	"database_catalogs":      (*ResourceDatabaseCatalog)(nil),
	"synced_database_tables": (*ResourceSyncedDatabaseTable)(nil),
	"alerts":                 (*ResourceAlert)(nil),
	"clusters":               (*ResourceCluster)(nil),
	"registered_models":      (*ResourceRegisteredModel)(nil),
	"dashboards":             (*ResourceDashboard)(nil),

	// Permissions
	"jobs.permissions":               (*ResourcePermissions)(nil),
	"pipelines.permissions":          (*ResourcePermissions)(nil),
	"apps.permissions":               (*ResourcePermissions)(nil),
	"clusters.permissions":           (*ResourcePermissions)(nil),
	"database_instances.permissions": (*ResourcePermissions)(nil),
	"experiments.permissions":        (*ResourcePermissions)(nil),
	"models.permissions":             (*ResourcePermissions)(nil),
	"sql_warehouses.permissions":     (*ResourcePermissions)(nil),
}

func InitAll(client *databricks.WorkspaceClient) (map[string]*Adapter, error) {
	result := make(map[string]*Adapter)
	for group, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, client)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", group, err)
		}
		result[group] = adapter
	}
	return result, nil
}
