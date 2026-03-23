package phases

// resourceDestroyMessage maps each resource type to a warning message displayed
// when the resource is deleted or recreated. An empty string means the resource
// is safe to delete/recreate without a warning.
//
// Rubric: a resource needs a warning if its deletion may cause non-recoverable
// data loss. A resource is safe if it holds only ephemeral state or configuration
// that is fully recoverable by redeploying the bundle.
//
// Every resource type returned by [config.SupportedResources] must have an entry.
// This is enforced by TestResourceDestroyMessageIsComplete.
var resourceDestroyMessage = map[string]string{
	// Safe resources (no warning):

	// Jobs: run history persists independently of the job definition.
	"jobs": "",
	// Model serving endpoints: stateless config; inference tables live in UC independently.
	"model_serving_endpoints": "",
	// Clusters: pure ephemeral compute; all config is in the bundle.
	"clusters": "",
	// Apps: stateless; all config and code deployed from bundle.
	"apps": "",
	// SQL warehouses: compute endpoint; query history stored separately.
	"sql_warehouses": "",
	// External locations: metadata pointer only; underlying cloud storage is not deleted.
	"external_locations": "",
	// Synced database tables: PurgeData=false preserves synced data; source always preserved.
	"synced_database_tables": "",
	// Postgres endpoints: stateless connection config; data lives in branch/project.
	"postgres_endpoints": "",

	// Unsafe resources (warning required):

	"schemas": "This action will result in the deletion or recreation of the following " +
		"UC schemas along with all the tables and views they contain:",

	"volumes": "This action will result in the deletion or recreation of the following volumes. " +
		"For managed volumes, the files stored in the volume are also deleted from your " +
		"cloud tenant within 30 days. For external volumes, the metadata about the volume " +
		"is removed from the catalog, but the underlying files are not deleted:",

	"pipelines": "This action will result in the deletion or recreation of the following " +
		"Lakeflow Spark Declarative Pipelines along with the Streaming Tables (STs) and " +
		"Materialized Views (MVs) managed by them:",

	"dashboards": "This action will result in the deletion or recreation of the following " +
		"dashboards. Recreated dashboards will have new IDs and permanent URLs:",

	"catalogs": "This action will result in the deletion or recreation of the following " +
		"UC catalogs along with all the schemas, tables, and views they contain:",

	"secret_scopes": "This action will result in the deletion or recreation of the following " +
		"secret scopes. All secrets stored in them will be permanently lost:",

	"database_instances": "This action will result in the deletion or recreation of the following " +
		"database instances. All data stored in them will be permanently lost:",

	"database_catalogs": "This action will result in the deletion or recreation of the following " +
		"database catalogs. All data stored in them will be permanently lost:",

	"postgres_projects": "This action will result in the deletion or recreation of the following " +
		"Postgres projects along with all their branches, databases, and endpoints:",

	"postgres_branches": "This action will result in the deletion or recreation of the following " +
		"Postgres branches along with all data they contain:",

	"models": "This action will result in the deletion or recreation of the following " +
		"MLflow models along with all their versions and artifacts:",

	"registered_models": "This action will result in the deletion or recreation of the following " +
		"registered models along with all their versions:",

	"experiments": "This action will result in the deletion or recreation of the following " +
		"MLflow experiments along with all their runs, metrics, and artifacts:",

	"quality_monitors": "This action will result in the deletion or recreation of the following " +
		"quality monitors. Associated metric tables may be lost or orphaned:",

	"alerts": "This action will result in the deletion or recreation of the following " +
		"alerts. Evaluation and notification history will be permanently lost:",
}
