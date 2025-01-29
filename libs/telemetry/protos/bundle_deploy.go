package protos

type BundleDeployEvent struct {
	// UUID associated with the bundle itself. Set in the `bundle.uuid` field in the bundle configuration.
	BundleUuid string `json:"bundle_uuid,omitempty"`

	ResourceCount                     int64 `json:"resource_count,omitempty"`
	ResourceJobCount                  int64 `json:"resource_job_count,omitempty"`
	ResourcePipelineCount             int64 `json:"resource_pipeline_count,omitempty"`
	ResourceModelCount                int64 `json:"resource_model_count,omitempty"`
	ResourceExperimentCount           int64 `json:"resource_experiment_count,omitempty"`
	ResourceModelServingEndpointCount int64 `json:"resource_model_serving_endpoint_count,omitempty"`
	ResourceRegisteredModelCount      int64 `json:"resource_registered_model_count,omitempty"`
	ResourceQualityMonitorCount       int64 `json:"resource_quality_monitor_count,omitempty"`
	ResourceSchemaCount               int64 `json:"resource_schema_count,omitempty"`
	ResourceVolumeCount               int64 `json:"resource_volume_count,omitempty"`
	ResourceClusterCount              int64 `json:"resource_cluster_count,omitempty"`
	ResourceDashboardCount            int64 `json:"resource_dashboard_count,omitempty"`
	ResourceAppCount                  int64 `json:"resource_app_count,omitempty"`

	// IDs of resources managed by the bundle. Some resources like volumes or schemas
	// do not expose a numerical or UUID identifier and are tracked by name. Those
	// resources are not tracked here since the names are PII.
	ResourceJobIDs       []string `json:"resource_job_ids,omitempty"`
	ResourcePipelineIDs  []string `json:"resource_pipeline_ids,omitempty"`
	ResourceClusterIDs   []string `json:"resource_cluster_ids,omitempty"`
	ResourceDashboardIDs []string `json:"resource_dashboard_ids,omitempty"`

	Experimental *BundleDeployExperimental `json:"experimental,omitempty"`
}

// These metrics are experimental and are often added in an adhoc manner. There
// are no guarantees for these metrics and they maybe removed in the future without
// any notice.
type BundleDeployExperimental struct {
	// Number of configuration files in the bundle.
	ConfigurationFileCount int64 `json:"configuration_file_count,omitempty"`

	// Size in bytes of the Terraform state file
	TerraformStateSizeBytes int64 `json:"terraform_state_size_bytes,omitempty"`

	// Number of variables in the bundle
	VariableCount        int64 `json:"variable_count,omitempty"`
	ComplexVariableCount int64 `json:"complex_variable_count,omitempty"`
	LookupVariableCount  int64 `json:"lookup_variable_count,omitempty"`

	// Number of targets in the bundle
	TargetCount int64 `json:"target_count,omitempty"`

	// Whether a field is set or not. If a configuration field is not present in this
	// map then it is not tracked by this field.
	// Keys are the full path of the field in the configuration tree.
	// Examples: "bundle.terraform.exec_path", "bundle.git.branch" etc.
	SetFields []BoolMapEntry `json:"set_fields,omitempty"`

	// Values for boolean configuration fields like `experimental.python_wheel_wrapper`
	// We don't need to define protos to track boolean values and can simply write those
	// values to this map to track them.
	BoolValues []BoolMapEntry `json:"bool_values,omitempty"`

	BundleMode BundleMode `json:"bundle_mode,omitempty"`

	WorkspaceArtifactPathType BundleDeployArtifactPathType `json:"workspace_artifact_path_type,omitempty"`

	// Execution time per mutator for a selected subset of mutators.
	BundleMutatorExecutionTimeMs []IntMapEntry `json:"bundle_mutator_execution_time_ms,omitempty"`
}

type BoolMapEntry struct {
	Key   string `json:"key,omitempty"`
	Value bool   `json:"value,omitempty"`
}

type IntMapEntry struct {
	Key   string `json:"key,omitempty"`
	Value int64  `json:"value,omitempty"`
}
