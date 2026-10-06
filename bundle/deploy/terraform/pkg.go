package terraform

const (
	TerraformConfigFileName = "bundle.tf.json"
)

var GroupToTerraformName = map[string]string{
	// 2 level groups: resources.GROUP
	"jobs":                    "databricks_job",
	"pipelines":               "databricks_pipeline",
	"models":                  "databricks_mlflow_model",
	"experiments":             "databricks_mlflow_experiment",
	"model_serving_endpoints": "databricks_model_serving",
	"registered_models":       "databricks_registered_model",
	"quality_monitors":        "databricks_quality_monitor",
	"schemas":                 "databricks_schema",
	"clusters":                "databricks_cluster",
	"dashboards":              "databricks_dashboard",
	"volumes":                 "databricks_volume",
	"apps":                    "databricks_app",
	"secret_scopes":           "databricks_secret_scope",
	"alerts":                  "databricks_alert_v2",
	"sql_warehouses":          "databricks_sql_endpoint",
	"database_instances":      "databricks_database_instance",
	"database_catalogs":       "databricks_database_database_catalog",
	"synced_database_tables":  "databricks_database_synced_database_table",
	"postgres_projects":       "databricks_postgres_project",
	"postgres_branches":       "databricks_postgres_branch",
	"postgres_databases":      "databricks_postgres_database",
	"postgres_endpoints":      "databricks_postgres_endpoint",
	"postgres_catalogs":       "databricks_postgres_catalog",
	"postgres_roles":          "databricks_postgres_role",
	"postgres_synced_tables":  "databricks_postgres_synced_table",

	// 3 level groups: resources.*.GROUP
	"permissions": "databricks_permissions",
	"grants":      "databricks_grants",
	"secret_acls": "databricks_secret_acl",
}

var TerraformToGroupName = func() map[string]string {
	m := make(map[string]string, len(GroupToTerraformName))
	for k, v := range GroupToTerraformName {
		m[v] = k
	}
	return m
}()
