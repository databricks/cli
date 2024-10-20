package config

const Paused = "PAUSED"
const Unpaused = "UNPAUSED"

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

	// Tags to add to all resources.
	Tags map[string]string `json:"tags,omitempty"`

	// Catalog is the default catalog for all resources.
	Catalog string `json:"catalog,omitempty"`

	// Schema is the default schema for all resources.
	Schema string `json:"schema,omitempty"`
}

// IsExplicitlyEnabled tests whether this feature is explicitly enabled.
func IsExplicitlyEnabled(feature *bool) bool {
	return feature != nil && *feature
}

// IsExplicitlyDisabled tests whether this feature is explicitly disabled.
func IsExplicitlyDisabled(feature *bool) bool {
	return feature != nil && !*feature
}
