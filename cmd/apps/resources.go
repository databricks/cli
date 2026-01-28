package apps

// Resource name constants for databricks.yml resource bindings.
// These names are used in resource.name fields in databricks.yml and must match
// the keys used in environment variable resolution.
const (
	ResourceNameSQLWarehouse    = "sql-warehouse"
	ResourceNameServingEndpoint = "serving-endpoint"
	ResourceNameExperiment      = "experiment"
	ResourceNameDatabase        = "database"
	ResourceNameDatabaseName    = "database-name"
	ResourceNameUCVolume        = "uc-volume"
)
