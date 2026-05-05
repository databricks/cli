package protos

type BundleDeployEvent struct {
	// UUID associated with the bundle itself. Set in the `bundle.uuid` field in the bundle configuration.
	BundleUuid string `json:"bundle_uuid,omitempty"`

	// UUID associated with the deployment.
	DeploymentId string `json:"deployment_id,omitempty"`

	// Error message encountered during the bundle deploy command, if any.
	ErrorMessage string `json:"error_message,omitempty"`

	// Deprecated: use Experimental.StateSize.ResourceStats[*].Count instead.
	// Per-resource-type counts are derived from deployment state at telemetry-
	// emission time and tracked under StateSize.
	ResourceCount                     int64 `json:"resource_count"`
	ResourceJobCount                  int64 `json:"resource_job_count"`
	ResourcePipelineCount             int64 `json:"resource_pipeline_count"`
	ResourceModelCount                int64 `json:"resource_model_count"`
	ResourceExperimentCount           int64 `json:"resource_experiment_count"`
	ResourceModelServingEndpointCount int64 `json:"resource_model_serving_endpoint_count"`
	ResourceRegisteredModelCount      int64 `json:"resource_registered_model_count"`
	ResourceQualityMonitorCount       int64 `json:"resource_quality_monitor_count"`
	ResourceSchemaCount               int64 `json:"resource_schema_count"`
	ResourceVolumeCount               int64 `json:"resource_volume_count"`
	ResourceClusterCount              int64 `json:"resource_cluster_count"`
	ResourceDashboardCount            int64 `json:"resource_dashboard_count"`
	ResourceAppCount                  int64 `json:"resource_app_count"`

	// IDs of resources managed by the bundle. Some resources like volumes or schemas
	// do not expose a numerical or UUID identifier and are tracked by name. Those
	// resources are not tracked here since the names are PII.
	ResourceJobIDs       []string `json:"resource_job_ids,omitempty"`
	ResourcePipelineIDs  []string `json:"resource_pipeline_ids,omitempty"`
	ResourceClusterIDs   []string `json:"resource_cluster_ids,omitempty"`
	ResourceDashboardIDs []string `json:"resource_dashboard_ids,omitempty"`

	// Per-resource-type metadata (counts and state-size statistics).
	ResourcesMetadata *BundleResourcesMetadata `json:"resources_metadata,omitempty"`

	Experimental *BundleDeployExperimental `json:"experimental,omitempty"`
}

// These metrics are experimental and are often added in an adhoc manner. There
// are no guarantees for these metrics and they maybe removed in the future without
// any notice.
type BundleDeployExperimental struct {
	// Number of configuration files in the bundle.
	ConfigurationFileCount int64 `json:"configuration_file_count"`

	// Size in bytes of the Terraform state file
	TerraformStateSizeBytes int64 `json:"terraform_state_size_bytes,omitempty"`

	// Number of variables in the bundle
	VariableCount        int64 `json:"variable_count"`
	ComplexVariableCount int64 `json:"complex_variable_count"`
	LookupVariableCount  int64 `json:"lookup_variable_count"`

	// Number of targets in the bundle
	TargetCount int64 `json:"target_count"`

	// Whether a field is set or not. If a configuration field is not present in this
	// map then it is not tracked by this field.
	// Keys are the full path of the field in the configuration tree.
	// Examples: "bundle.terraform.exec_path", "bundle.git.branch" etc.
	SetFields []BoolMapEntry `json:"set_fields,omitempty"`

	// Values for boolean configuration fields like `experimental.python_wheel_wrapper` or just any
	// boolean values that we want to track.
	// We don't need to define protos to track boolean values and can simply write those
	// values to this map to track them.
	BoolValues []BoolMapEntry `json:"bool_values,omitempty"`

	BundleMode BundleMode `json:"bundle_mode,omitempty"`

	WorkspaceArtifactPathType BundleDeployArtifactPathType `json:"workspace_artifact_path_type,omitempty"`

	// Execution time per mutator for a selected subset of mutators.
	BundleMutatorExecutionTimeMs []IntMapEntry `json:"bundle_mutator_execution_time_ms,omitempty"`

	// Number of resources added by PythonMutator
	PythonAddedResourcesCount int64 `json:"python_added_resources_count,omitempty"`

	// Number of resources updated by PythonMutator
	PythonUpdatedResourcesCount int64 `json:"python_updated_resources_count,omitempty"`

	// Number of resource loaders declared at 'python/resources' in databricks.yml
	PythonResourceLoadersCount int64 `json:"python_resource_loaders_count,omitempty"`

	// Number of resource mutators declared at 'python/mutators' in databricks.yml
	PythonResourceMutatorsCount int64 `json:"python_resource_mutators_count,omitempty"`

	// Local cache measurements in milliseconds (compute duration, potential savings, etc.)
	LocalCacheMeasurementsMs []IntMapEntry `json:"local_cache_measurements_ms,omitempty"`
}

// BundleResourcesMetadata mirrors the universe proto. Per-resource-type
// metadata for one bundle deployment, including counts (which replace the
// deprecated DatabricksBundleDeployEvent.resource_*_count fields) and
// state-size statistics.
type BundleResourcesMetadata struct {
	// "direct" or "terraform"
	StateEngine string `json:"state_engine,omitempty"`

	// Size in bytes of the entire deployment state file on disk.
	StateFileSizeBytes int64 `json:"state_file_size_bytes,omitempty"`

	// One entry per resource type present in the bundle.
	Resources []ResourceMetadata `json:"resources,omitempty"`
}

// ResourceMetadata holds metadata about resources of a single type within one
// bundle deployment.
type ResourceMetadata struct {
	// Resource type name: "jobs", "pipelines", "schemas", ...
	ResourceType string `json:"resource_type,omitempty"`

	// Number of resources of this type declared in the bundle configuration.
	Count int64 `json:"count,omitempty"`

	// State-size statistics across resources of this type tracked in the
	// deployment state. Zero when no resources of this type are in state.
	StateSizeMaxBytes    int64 `json:"state_size_max_bytes,omitempty"`
	StateSizeMeanBytes   int64 `json:"state_size_mean_bytes,omitempty"`
	StateSizeMedianBytes int64 `json:"state_size_median_bytes,omitempty"`
}

type BoolMapEntry struct {
	Key   string `json:"key,omitempty"`
	Value bool   `json:"value"`
}

type IntMapEntry struct {
	Key   string `json:"key,omitempty"`
	Value int64  `json:"value"`
}
