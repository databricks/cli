package terraform

import "strings"

// BindOptions configures binding a bundle resource to an existing remote resource.
type BindOptions struct {
	AutoApprove  bool
	ResourceType string
	ResourceKey  string
	ResourceId   string
}

var prefixToGroup = []struct{ prefix, group string }{
	{"job_", "jobs"},
	{"pipeline_", "pipelines"},
	{"mlflow_experiment_", "experiments"},
	{"mlflow_model_", "models"},
	{"cluster_", "clusters"},
	{"app_", "apps"},
	{"dashboard_", "dashboards"},
	{"alert_", "alerts"},
	{"model_serving_", "model_serving_endpoints"},
	{"sql_endpoint_", "sql_warehouses"},
	{"database_instance_", "database_instances"},
	{"postgres_project_", "postgres_projects"},
}

var grantsPrefix = []struct{ prefix, group string }{
	{"schema_", "schemas"},
	{"volume_", "volumes"},
	{"registered_model_", "registered_models"},
}

// convertPermissionsResourceNameToKey converts terraform permission resource names back to hierarchical resource keys
// e.g., "mlflow_experiment_foo" -> "resources.experiments.foo.permissions"
func convertPermissionsResourceNameToKey(terraformName string) string {
	for _, pg := range prefixToGroup {
		if resourceName, found := strings.CutPrefix(terraformName, pg.prefix); found {
			return "resources." + pg.group + "." + resourceName + ".permissions"
		}
	}

	// Fallback: if no known prefix is found, use the old behavior
	return ""
}

// convertGrantsResourceNameToKey converts terraform grants resource names back to hierarchical resource keys
// e.g., "schema_foo" -> "resources.schemas.foo.grants"
func convertGrantsResourceNameToKey(terraformName string) string {
	for _, gp := range grantsPrefix {
		if resourceName, found := strings.CutPrefix(terraformName, gp.prefix); found {
			return "resources." + gp.group + "." + resourceName + ".grants"
		}
	}

	// Fallback: if no known prefix is found, use the old behavior
	return ""
}
