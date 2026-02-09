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
	ResourceTypeDatabase          = "database"
	ResourceTypeGenieSpace        = "genie_space"
	ResourceTypeExperiment        = "experiment"
	ResourceTypeApp               = "app"
)

// PromptResourceFunc prompts the user for a resource of a given type and returns the selected ID.
type PromptResourceFunc func(ctx context.Context, r manifest.Resource, required bool) (string, error)

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
	case ResourceTypeGenieSpace:
		return PromptForGenieSpace, true
	case ResourceTypeExperiment:
		return PromptForExperiment, true
	case ResourceTypeApp:
		return PromptForAppResource, true
	default:
		return nil, false
	}
}
