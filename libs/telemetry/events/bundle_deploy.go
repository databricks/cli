package events

import (
	"encoding/json"
	"sort"
)

type BundleDeployEvent struct {
	Configuration BundleConfiguration `json:"configuration,omitempty"`
	Metrics       BundleDeployMetrics `json:"metrics,omitempty"`
	State         BundleState         `json:"state,omitempty"`
}

type BundleDeployMetrics struct {
	// Number of variables in the bundle configuration
	VariableCount        int64 `json:"variable_count"`
	ComplexVariableCount int64 `json:"complex_variable_count"`
	LookupVariableCount  int64 `json:"lookup_variable_count"`

	// Number of resources in the bundle configuration
	ResourceCount             int64 `json:"resource_count"`
	JobCount                  int64 `json:"job_count"`
	PipelineCount             int64 `json:"pipeline_count"`
	ModelCount                int64 `json:"model_count"`
	ExperimentCount           int64 `json:"experiment_count"`
	ModelServingEndpointCount int64 `json:"model_serving_endpoint_count"`
	RegisteredModelCount      int64 `json:"registered_model_count"`
	QualityMonitorCount       int64 `json:"quality_monitor_count"`
	SchemaCount               int64 `json:"schema_count"`
	VolumeCount               int64 `json:"volume_count"`
	ClusterCount              int64 `json:"cluster_count"`
	DashboardCount            int64 `json:"dashboard_count"`

	// Number of YAML configuration files in the bundle configuration.
	ConfigurationFileCount int64 `json:"configuration_file_count"`

	// Size in bytes of the Terraform state file
	TfstateBytes int64 `json:"tfstate_bytes"`

	// Total execution time of the bundle deployment in milliseconds
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// Execution time per mutator for a selected subset of mutators.
	MutatorExecutionTimeMs []IntMapEntry `json:"mutator_execution_time_ms,omitempty"`

	PreinitScriptCount    int64 `json:"preinit_script_count"`
	PostinitScriptCount   int64 `json:"postinit_script_count"`
	PrebuildScriptCount   int64 `json:"prebuild_script_count"`
	PostbuildScriptCount  int64 `json:"postbuild_script_count"`
	PredeployScriptCount  int64 `json:"predeploy_script_count"`
	PostdeployScriptCount int64 `json:"postdeploy_script_count"`

	// Number of targets in the bundle
	TargetCount int64 `json:"target_count"`
}

type IntMapEntry struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type BoolMapEntry struct {
	Key   string `json:"key"`
	Value bool   `json:"value"`
}

type BoolMap struct{ m []BoolMapEntry }

func (b *BoolMap) Append(key string, value bool) {
	if b.m == nil {
		b.m = make([]BoolMapEntry, 0)
	}

	b.m = append(b.m, BoolMapEntry{Key: key, Value: value})
}

func (b *BoolMap) MarshalJSON() ([]byte, error) {
	sort.Slice(b.m, func(i, j int) bool {
		return b.m[i].Key < b.m[j].Key
	})

	return json.Marshal(b.m)
}

type BundleConfiguration struct {
	// Whether a field is set or not. If a configuration field is not present in this
	// map then it is not tracked by telemetry.
	// Keys are the full path of the field in the configuration tree.
	// Examples: "bundle.terraform.exec_path", "bundle.git.branch" etc.
	SetFields BoolMap `json:"set_fields,omitempty"`

	// Values for boolean configuration fields like `experimental.python_wheel_wrapper`
	// We don't need to define protos to track these values and can simply write those
	// values to this field to track them.
	BoolValues BoolMap `json:"bool_values,omitempty"`

	BundleUuid       string           `json:"bundle_uuid,omitempty"`
	BundleMode       BundleMode       `json:"bundle_mode,omitempty"`
	ArtifactPathType ArtifactPathType `json:"artifact_path_type,omitempty"`
}

type BundleMode string

const (
	BundleModeDevelopment BundleMode = "DEVELOPMENT"
	BundleModeProduction  BundleMode = "PRODUCTION"
)

type ArtifactPathType string

const (
	ArtifactPathTypeWorkspaceFilesystem ArtifactPathType = "WORKSPACE_FILESYSTEM"
	ArtifactPathTypeUCVolume            ArtifactPathType = "UC_VOLUME"
)

type BundleState struct {
	// IDs of resources managed by the bundle. Some resources like Volumes or Schemas
	// do not expose a numerical or UUID identifier and are tracked by name. Those
	// resources are not tracked here since the names are PII.
	JobIds       []int64  `json:"job_ids,omitempty"`
	PipelinesIds []string `json:"pipeline_ids,omitempty"`
	ClusterIds   []int64  `json:"cluster_ids,omitempty"`
	DashboardIds []string `json:"dashboard_ids,omitempty"`
}
