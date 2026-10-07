package resourcemutator

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// When a new resource is added to configuration, we apply bundle
// settings and defaults to it. Initialization is applied only once.
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyInitializeMutators(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplySeqContext(
		ctx,
		b,
		// Reads (typed): b.Config.RunAs, b.Config.Workspace.CurrentUser (validates run_as configuration)
		// Reads (dynamic): run_as (checks if run_as is specified)
		// Updates (typed): b.Config.Resources.Jobs[].RunAs (sets job run_as fields to bundle run_as; only if Experimental.UseLegacyRunAs is set)
		// Updates (typed): range b.Config.Resources.Pipelines[].Permissions (set permission based on bundle run_as; only if Experimental.UseLegacyRunAs is set)
		SetRunAs(),

		// Reads (typed): b.Config.Bundle.{Mode,ClusterId} (checks mode and cluster ID settings)
		// Reads (env): DATABRICKS_CLUSTER_ID (environment variable for backward compatibility)
		// Reads (typed): b.Config.Resources.Jobs.*.Tasks.*.ForEachTask
		// Updates (typed): b.Config.Bundle.ClusterId (sets from environment if in development mode)
		// Updates (typed): b.Config.Resources.Jobs.*.Tasks.*.{NewCluster,ExistingClusterId,JobClusterKey,EnvironmentKey} (replaces compute settings with specified cluster ID)
		// OR corresponding fields on ForEachTask if that is present
		// Overrides job compute settings with a specified cluster ID for development or testing
		OverrideCompute(),

		// ApplyPresets should have more priority than defaults below, so it should be run first
		ApplyPresets(),
	)

	if logdiag.HasError(ctx) {
		return
	}

	defaults := []bundle.Default{
		{Pattern: "resources.dashboards.*.parent_path", Value: b.Config.Workspace.ResourcePath},
		{Pattern: "resources.dashboards.*.embed_credentials", Value: false},
		{Pattern: "resources.genie_spaces.*.parent_path", Value: b.Config.Workspace.ResourcePath},
		{Pattern: "resources.volumes.*.volume_type", Value: "MANAGED"},

		{Pattern: "resources.alerts.*.parent_path", Value: b.Config.Workspace.ResourcePath},

		// Jobs:

		// The defaults are the same as for terraform provider latest version (v1.75.0)
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/jobs/resource_job.go#L532
		{Pattern: "resources.jobs.*.name", Value: "Untitled"},
		{Pattern: "resources.jobs.*.max_concurrent_runs", Value: 1},
		{Pattern: "resources.jobs.*.schedule.pause_status", Value: "UNPAUSED"},
		{Pattern: "resources.jobs.*.trigger.pause_status", Value: "UNPAUSED"},
		{Pattern: "resources.jobs.*.continuous.pause_status", Value: "UNPAUSED"},

		// Enable queueing for jobs by default, following the behavior from API 2.2+.
		// As of 2024-04, we're still using API 2.1 which has queueing disabled by default.
		{Pattern: "resources.jobs.*.queue", Value: jobs.QueueSettings{Enabled: true}},

		// This is converted from single-task to multi-task
		{Pattern: "resources.jobs.*.task[*].dbt_task.schema", Value: "default"},
		{Pattern: "resources.jobs.*.task[*].for_each_task.task.dbt_task.schema", Value: "default"},

		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go
		{Pattern: "resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients.notebooks", Value: true},
		{Pattern: "resources.jobs.*.job_clusters[*].new_cluster.workload_type.clients.jobs", Value: true},

		// Pipelines (same as terraform)
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/pipelines/resource_pipeline.go#L253
		{Pattern: "resources.pipelines.*.edition", Value: "ADVANCED"},
		{Pattern: "resources.pipelines.*.channel", Value: "CURRENT"},

		// SqlWarehouses (same as terraform)
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/sql/resource_sql_endpoint.go#L59
		{Pattern: "resources.sql_warehouses.*.auto_stop_mins", Value: 120},
		{Pattern: "resources.sql_warehouses.*.enable_photon", Value: true},
		{Pattern: "resources.sql_warehouses.*.max_num_clusters", Value: 1},
		{Pattern: "resources.sql_warehouses.*.spot_instance_policy", Value: "COST_OPTIMIZED"},

		// Apps:
		{Pattern: "resources.apps.*.description", Value: ""},

		// Clusters (same as terraform)
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go#L315
		{Pattern: "resources.clusters.*.autotermination_minutes", Value: 60},
		{Pattern: "resources.clusters.*.workload_type.clients.notebooks", Value: true},
		{Pattern: "resources.clusters.*.workload_type.clients.jobs", Value: true},
	}

	bundle.SetDefaults(ctx, b, defaults)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		// Reads (typed): b.Config.Resources.Dashboards (checks dashboard configurations)
		// Updates (typed): b.Config.Resources.Dashboards[].ParentPath (ensures /Workspace prefix is present)
		// Ensures dashboard parent paths have the required /Workspace prefix
		DashboardFixups(),

		// Reads (typed): b.Config.Resources.GenieSpaces (checks genie space configurations)
		// Updates (typed): b.Config.Resources.GenieSpaces[].ParentPath (ensures /Workspace prefix is present)
		// Ensures genie space parent paths have the required /Workspace prefix
		GenieSpaceFixups(),

		// Reads (typed): b.Config.Permissions (validates permission levels)
		// Reads (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps,vector_search_endpoints,...}.*.permissions (reads existing permissions)
		// Updates (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps,vector_search_endpoints,...}.*.permissions (adds permissions from bundle-level configuration)
		// Applies bundle-level permissions to all supported resources
		ApplyBundlePermissions(),

		// Reads (typed): b.Config.Workspace.CurrentUser.UserName (gets current user name)
		// Updates (dynamic): resources.*.*.permissions
		FixPermissions(),
	)
}

// Normalization is applied multiple times if resource is modified during initialization
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyNormalizeMutators(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplySeqContext(
		ctx,
		b,

		validate.SingleNodeCluster(),

		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): resources.* (strings) (resolves variable references to their actual values)
		// Resolves variable references in 'resources' using bundle, workspace, and variables prefixes
		mutator.ResolveVariableReferencesOnlyResources(),

		// Reads (dynamic): resources.pipelines.*.libraries (checks for notebook.path and file.path fields)
		// Updates (dynamic): resources.pipelines.*.libraries (expands glob patterns in path fields to multiple library entries)
		// Expands glob patterns in pipeline library paths to include all matching files
		ExpandPipelineGlobPaths(),

		// Reads (dynamic): resources.jobs.*.job_clusters (reads job clusters to merge)
		// Updates (dynamic): resources.jobs.*.job_clusters (merges job clusters with the same job_cluster_key)
		MergeJobClusters(),

		// Reads (dynamic): resources.jobs.*.parameters (reads job parameters to merge)
		// Updates (dynamic): resources.jobs.*.parameters (merges job parameters with the same name)
		MergeJobParameters(),

		// Reads (dynamic): resources.jobs.*.tasks (reads job tasks to merge)
		// Updates (dynamic): resources.jobs.*.tasks (merges job tasks with the same task_key)
		MergeJobTasks(),

		// Reads (dynamic): resources.pipelines.*.clusters (reads pipeline clusters to merge)
		// Updates (dynamic): resources.pipelines.*.clusters (merges pipeline clusters with the same label)
		MergePipelineClusters(),

		// Reads (dynamic): resources.apps.*.resources (reads app resources to merge)
		// Updates (dynamic): resources.apps.*.resources (merges app resources with the same name)
		MergeApps(),

		// Reads (dynamic): resources.{catalogs,schemas,external_locations,volumes,registered_models,vector_search_indexes}.*.grants
		// Updates (dynamic): same paths — merges grant entries by principal and deduplicates privileges
		MergeGrants(),

		// Reads (typed): resources.{volumes,registered_models,pipelines,quality_monitors,model_serving_endpoints}.*.{catalog_name,schema_name,...}
		// Updates (typed): same paths — converts implicit schema/catalog references to explicit ${resources.schemas/catalogs.<key>.name} syntax
		// Also updates: resources.schemas.*.catalog_name (catalog dependency for schemas)
		// Translates implicit schema and catalog references across all UC resources to explicit syntax to capture dependencies
		CaptureUCDependencies(),

		// Reads (dynamic): resources.dashboards.*.file_path
		// Updates (dynamic): resources.dashboards.*.serialized_dashboard
		// Drops (dynamic): resources.dashboards.*.file_path
		ConfigureDashboardSerializedDashboard(),

		// Reads (dynamic): resources.genie_spaces.*.file_path
		// Updates (dynamic): resources.genie_spaces.*.serialized_space
		ConfigureGenieSpaceSerializedSpace(),

		// Reads (dynamic): resources.cluster_policies.*.definition
		// Updates (dynamic): resources.cluster_policies.*.definition (inline YAML -> JSON string)
		ConfigureClusterPolicyDefinition(),

		// Reads (typed): resources.alerts.*.file_path
		// Updates (typed): resources.alerts.* (loads alert configuration from .dbalert.json file)
		mutator.LoadDBAlertFiles(),

		// Reads and updates (typed): resources.jobs.*.**
		JobClustersFixups(),
		ClusterFixups(),

		// Reads (typed): resources.model_serving_endpoints.*.config.{served_models,served_entities}
		// Validates: Cannot use both served_models and served_entities at the same time
		// Updates (typed): resources.model_serving_endpoints.*.config.served_entities (converts served_models to served_entities)
		// Updates (typed): resources.model_serving_endpoints.*.config.served_entities[*].workload_size (sets default "Small" if not specified)
		// Drops: resources.model_serving_endpoints.*.config.served_models (after conversion)
		ModelServingEndpointFixups(),
	)
}

// NormalizeAndInitializeResources initializes and normalizes specified resources,
// and should be used by mutators after they have added resources.
func NormalizeAndInitializeResources(
	ctx context.Context,
	b *bundle.Bundle,
	addedResources ResourceKeySet,
) {
	if addedResources.IsEmpty() {
		return
	}

	restore := selectResources(b, addedResources)

	applyNormalizeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	applyInitializeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	// after mutators, we merge updated resources back to the snapshot to preserve non-selected resources
	err := restore()
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to merge resources: %w", err))
	}
}

// NormalizeResources normalizes resources specified resources,
// and should be used by mutators after they have updated resources.
func NormalizeResources(
	ctx context.Context,
	b *bundle.Bundle,
	updatedResources ResourceKeySet,
) {
	if updatedResources.IsEmpty() {
		return
	}

	restore := selectResources(b, updatedResources)

	applyNormalizeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	// Permissions added to an existing resource by a Python mutator must still get the
	// deploying user as IS_OWNER, otherwise the Permissions API rejects the PUT with
	// "must have exactly one owner" (#5682). FixPermissions is idempotent, so re-running
	// it on resources that already have an owner is a no-op. ApplyBundlePermissions is
	// intentionally not re-run here: it is not idempotent (it appends bundle-level
	// permissions) and already ran for these resources in ProcessStaticResources.
	bundle.ApplyContext(ctx, b, FixPermissions())
	if logdiag.HasError(ctx) {
		return
	}

	// after mutators, we merge updated resources back to the snapshot to preserve non-selected resources
	err := restore()
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to merge resources: %w", err))
	}
}

// resourceTypeFields returns the fields of resources that hold the resources of each type,
// by the name of the type (e.g. "jobs").
func resourceTypeFields(resources reflect.Value) map[string]reflect.Value {
	fields := map[string]reflect.Value{}
	t := resources.Type()
	for i := range t.NumField() {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" || resources.Field(i).Kind() != reflect.Map {
			continue
		}
		fields[name] = resources.Field(i)
	}
	return fields
}

// selectResources removes all resources except the ones in resourceKeys from the bundle
// configuration. It returns a function that restores the removed resources: it puts the
// configuration back to what it was before, except for the selected resources, which take
// the values they have at that point. Any other changes to the configuration are discarded.
func selectResources(b *bundle.Bundle, resourceKeys ResourceKeySet) func() error {
	// A shallow copy of the configuration: the resources maps are replaced below, not
	// modified, so the copy keeps all resources.
	snapshot := b.Config

	for resourceType, field := range resourceTypeFields(reflect.ValueOf(&b.Config.Resources).Elem()) {
		if _, ok := resourceKeys[resourceType]; !ok {
			field.SetZero()
			continue
		}

		selected := reflect.MakeMap(field.Type())
		for _, name := range resourceKeys.Names(resourceType) {
			v := field.MapIndex(reflect.ValueOf(name))
			if v.IsValid() {
				selected.SetMapIndex(reflect.ValueOf(name), v)
			}
		}
		field.Set(selected)
	}

	return func() error {
		updated := b.Config
		updatedView := updated.View()
		updatedFields := resourceTypeFields(reflect.ValueOf(&updated.Resources).Elem())

		b.Config = snapshot
		fields := resourceTypeFields(reflect.ValueOf(&b.Config.Resources).Elem())

		for resourceType, names := range resourceKeys {
			for name := range names {
				v := updatedFields[resourceType].MapIndex(reflect.ValueOf(name))
				if !v.IsValid() {
					continue
				}

				path := structpath.NewStringKeys(nil, "resources", resourceType, name)
				err := b.Config.Assign(path, updatedView.Lookup(path))
				if err != nil {
					return err
				}

				// Keep the resource that was updated, it may have more than the configuration describes.
				field := fields[resourceType]
				if field.IsNil() {
					field.Set(reflect.MakeMap(field.Type()))
				}
				field.SetMapIndex(reflect.ValueOf(name), v)
			}
		}
		return nil
	}
}
