package prompt

import (
	"context"

	"github.com/databricks/cli/libs/apps/manifest"
)

// Resource type constants matching the TS enum (appkit plugin manifest).
const (
	ResourceTypeSecret            = "secret"
	ResourceTypeJob               = "job"
	ResourceTypeSQLWarehouse      = "sql_warehouse"
	ResourceTypeServingEndpoint   = "serving_endpoint"
	ResourceTypeVolume            = "volume"
	ResourceTypeVectorSearchIndex = "vector_search_index"
	ResourceTypeUCFunction        = "uc_function"
	ResourceTypeUCConnection      = "uc_connection"
	ResourceTypeDatabase          = "database" // Lakebase Provisioned (V1)
	ResourceTypePostgres          = "postgres" // Lakebase Autoscaling (V2)
	ResourceTypeGenieSpace        = "genie_space"
	ResourceTypeExperiment        = "experiment"
	// TODO: uncomment when bundles support app as an app resource type.
	// ResourceTypeApp = "app"
)

// PromptResourceFunc prompts the user for a resource of a given type.
// Returns a map of value keys to values. For single-field resources the map has one entry
// keyed by "resource_key.field" (e.g., {"sql-warehouse.id": "abc123"}). For multi-field resources,
// keys use the format "resource_key.field_name" (e.g., {"database.instance_name": "x", "database.database_name": "y"}).
type PromptResourceFunc func(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error)

// GetPromptFunc returns the prompt function for the given resource type, or (nil, false) if not supported.
func GetPromptFunc(resourceType string) (PromptResourceFunc, bool) {
	switch resourceType {
	case ResourceTypeSecret:
		return PromptForSecret, true
	case ResourceTypeJob:
		return PromptForJob, true
	case ResourceTypeSQLWarehouse:
		return PromptForSQLWarehouseResource, true
	case ResourceTypeServingEndpoint:
		return PromptForServingEndpoint, true
	case ResourceTypeVolume:
		return PromptForVolume, true
	case ResourceTypeVectorSearchIndex:
		return PromptForVectorSearchIndex, true
	case ResourceTypeUCFunction:
		return PromptForUCFunction, true
	case ResourceTypeUCConnection:
		return PromptForUCConnection, true
	case ResourceTypeDatabase:
		return PromptForDatabase, true
	case ResourceTypePostgres:
		return PromptForPostgres, true
	case ResourceTypeGenieSpace:
		return PromptForGenieSpace, true
	case ResourceTypeExperiment:
		return PromptForExperiment, true
	// TODO: uncomment when bundles support app as an app resource type.
	// case ResourceTypeApp:
	// 	return PromptForAppResource, true
	default:
		return nil, false
	}
}
