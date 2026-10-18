// Package resourcedefaults lists the values set on resources where the configuration
// does not set them. The walker that applies them is generated from the patterns
// (bundle/config/mutator/resourcemutator/defaults_walk.go).
package resourcedefaults

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// Default is a value to set at every path matching Pattern where it is not set yet.
type Default struct {
	Pattern string
	Value   func(r *config.Root) any
}

func constant(v any) func(*config.Root) any {
	return func(*config.Root) any { return v }
}

func resourcePath(r *config.Root) any {
	return r.Workspace.ResourcePath
}

// Defaults are applied in order.
var Defaults = []Default{
	{Pattern: "resources.dashboards.*.parent_path", Value: resourcePath},
	{Pattern: "resources.dashboards.*.embed_credentials", Value: constant(false)},
	{Pattern: "resources.genie_spaces.*.parent_path", Value: resourcePath},
	{Pattern: "resources.volumes.*.volume_type", Value: constant("MANAGED")},

	{Pattern: "resources.alerts.*.parent_path", Value: resourcePath},

	// Jobs:

	// The defaults are the same as for terraform provider latest version (v1.75.0)
	// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/jobs/resource_job.go#L532
	{Pattern: "resources.jobs.*.name", Value: constant("Untitled")},
	{Pattern: "resources.jobs.*.max_concurrent_runs", Value: constant(1)},
	{Pattern: "resources.jobs.*.schedule.pause_status", Value: constant("UNPAUSED")},
	{Pattern: "resources.jobs.*.trigger.pause_status", Value: constant("UNPAUSED")},
	{Pattern: "resources.jobs.*.continuous.pause_status", Value: constant("UNPAUSED")},

	// Enable queueing for jobs by default, following the behavior from API 2.2+.
	// As of 2024-04, we're still using API 2.1 which has queueing disabled by default.
	{Pattern: "resources.jobs.*.queue", Value: constant(jobs.QueueSettings{Enabled: true})},

	// This is converted from single-task to multi-task. The dbt_task.schema default
	// ("default") was declared at resources.jobs.*.task[*], which is not a field, so it
	// never applied; #7097 fixes the pattern on main.

	// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go
	{Pattern: "resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients.notebooks", Value: constant(true)},
	{Pattern: "resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients.jobs", Value: constant(true)},

	// Pipelines (same as terraform)
	// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/pipelines/resource_pipeline.go#L253
	{Pattern: "resources.pipelines.*.edition", Value: constant("ADVANCED")},
	{Pattern: "resources.pipelines.*.channel", Value: constant("CURRENT")},

	// SqlWarehouses (same as terraform)
	// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/sql/resource_sql_endpoint.go#L59
	{Pattern: "resources.sql_warehouses.*.auto_stop_mins", Value: constant(120)},
	{Pattern: "resources.sql_warehouses.*.enable_photon", Value: constant(true)},
	{Pattern: "resources.sql_warehouses.*.max_num_clusters", Value: constant(1)},
	{Pattern: "resources.sql_warehouses.*.spot_instance_policy", Value: constant("COST_OPTIMIZED")},

	// Apps:
	{Pattern: "resources.apps.*.description", Value: constant("")},

	// Clusters (same as terraform)
	// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go#L315
	{Pattern: "resources.clusters.*.autotermination_minutes", Value: constant(60)},
	{Pattern: "resources.clusters.*.workload_type.clients.notebooks", Value: constant(true)},
	{Pattern: "resources.clusters.*.workload_type.clients.jobs", Value: constant(true)},
}

// Patterns returns the patterns of Defaults, in order.
func Patterns() []string {
	out := make([]string, len(Defaults))
	for i, d := range Defaults {
		out[i] = d.Pattern
	}
	return out
}
