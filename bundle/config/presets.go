package config

const (
	Paused   = "PAUSED"
	Unpaused = "UNPAUSED"
)

type Presets struct {
	// NamePrefix to prepend to all resource names.
	NamePrefix string `json:"name_prefix,omitempty"`

	// PipelinesDevelopment is the default value for the development field of pipelines.
	PipelinesDevelopment *bool `json:"pipelines_development,omitempty"`

	// TriggerPauseStatus is the default value for the pause status of all triggers and schedules.
	// Either config.Paused, config.Unpaused, or empty.
	TriggerPauseStatus string `json:"trigger_pause_status,omitempty"`

	// JobsMaxConcurrentRuns is the default value for the max concurrent runs of jobs.
	JobsMaxConcurrentRuns int `json:"jobs_max_concurrent_runs,omitempty"`

	// SourceLinkedDeployment indicates whether source-linked deployment is enabled. Works only in Databricks Workspace
	// When set to true, resources created during deployment will point to source files in the workspace instead of their workspace copies.
	// File synchronization to ${workspace.file_path} is skipped.
	SourceLinkedDeployment *bool `json:"source_linked_deployment,omitempty"`

	// ArtifactsDynamicVersion enables dynamic_version on every artifact.
	ArtifactsDynamicVersion bool `json:"artifacts_dynamic_version,omitempty"`

	// Tags to add to all resources.
	Tags map[string]string `json:"tags,omitempty"`
}

// IsExplicitlyEnabled tests whether this feature is explicitly enabled.
func IsExplicitlyEnabled(feature *bool) bool {
	return feature != nil && *feature
}

// IsExplicitlyDisabled tests whether this feature is explicitly disabled.
func IsExplicitlyDisabled(feature *bool) bool {
	return feature != nil && !*feature
}
