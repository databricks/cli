package dresources

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

var SupportedResources = map[string]any{
	"jobs":                   (*ResourceJob)(nil),
	"jobs.permissions":       (*ResourcePermissions)(nil),
	"pipelines":              (*ResourcePipeline)(nil),
	"pipelines.permissions":  (*ResourcePermissions)(nil),
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
