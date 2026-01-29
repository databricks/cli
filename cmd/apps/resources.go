package apps

import "github.com/databricks/cli/cmd/apps/internal"

// Resource name constants for databricks.yml resource bindings.
// Re-exported from internal/appflags for convenience.
const (
	ResourceNameSQLWarehouse    = internal.ResourceNameSQLWarehouse
	ResourceNameServingEndpoint = internal.ResourceNameServingEndpoint
	ResourceNameExperiment      = internal.ResourceNameExperiment
	ResourceNameDatabase        = internal.ResourceNameDatabase
	ResourceNameDatabaseName    = internal.ResourceNameDatabaseName
	ResourceNameUCVolume        = internal.ResourceNameUCVolume
)
