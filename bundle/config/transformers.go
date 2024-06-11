package config

type Prefix struct {
	Value string `json:"value,omitempty"`
	// Whether this feature is enabled. Treated as 'true' when not set.
	Enabled *bool `json:"enabled,omitempty"`
}

type PipelinesDevelopment struct {
	// Whether this feature is enabled. Treated as 'true' when not set.
	Enabled *bool `json:"enabled,omitempty"`
}

type TriggerPauseStatus struct {
	// Whether to set PauseStatus to PauseStatusPaused
	Enabled *bool `json:"enabled,omitempty"`
}

type JobsMaxConcurrentRuns struct {
	Value int `json:"value,omitempty"`
	// Whether this feature is enabled. Treated as 'true' when not set.
	Enabled *bool `json:"enabled,omitempty"`
}

type Tags struct {
	Tags map[string]string `json:"tags,omitempty"`
	// Whether this feature is enabled. Treated as 'true' when not set.
	Enabled *bool `json:"enabled,omitempty"`
}

type Transformers struct {
	Prefix                Prefix                `json:"prefix,omitempty"`
	PipelinesDevelopment  PipelinesDevelopment  `json:"pipelines_set_development,omitempty"`
	TriggerPauseStatus    TriggerPauseStatus    `json:"trigger_pause_status,omitempty"`
	JobsMaxConcurrentRuns JobsMaxConcurrentRuns `json:"jobs_max_concurrent_runs,omitempty"`
	Tags                  Tags                  `json:"tags,omitempty"`
}
