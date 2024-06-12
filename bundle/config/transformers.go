package config

type Transformers struct {
	// Prefix to prepend to all resource names.
	Prefix string `json:"prefix,omitempty"`

	// DefaultPipelinesDevelopment is the default value for the development field of pipelines.
	DefaultPipelinesDevelopment *bool `json:"default_pipelines_development,omitempty"`

	// DefaultTriggerPauseStatus is the default value for the pause status of all triggers and schedules.
	DefaultTriggerPauseStatus *bool `json:"default_trigger_pause_status,omitempty"`

	// DefaultJobsMaxConcurrentRuns is the default value for the max concurrent runs of jobs.
	DefaultJobsMaxConcurrentRuns int `json:"default_jobs_max_concurrent_runs,omitempty"`

	// Tags to add to all resources.
	Tags *map[string]string `json:"tags,omitempty"`
}

// IsExplicitlyEnabled tests whether this feature is explicitly enabled.
func IsExplicitlyEnabled(feature *bool) bool {
	return feature != nil && *feature
}

// IsExplicitlyDisabled tests whether this feature is explicitly disabled.
func IsExplicitlyDisabled(feature *bool) bool {
	return feature != nil && !*feature
}
