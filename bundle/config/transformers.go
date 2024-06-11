package config

// A feature with an enablement flag that is on by default.
type EnableableDefaultOn struct {
	// Enabled is a user-configurable value controlling enablement
	// Use IsEnabled to read this value.
	Enabled *bool `json:"enabled,omitempty"`
}

// IsEnabled tests whether this feature is enabled.
// EnableableDefaultOn features are considered enabled by default.
func (e EnableableDefaultOn) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}

// IsExplicitlyDisabled tests whether this feature is explicitly disabled.
func (e EnableableDefaultOff) IsExplicitlyDisabled() bool {
	return e.Enabled != nil && !*e.Enabled
}

// A feature with an enablement flag that is off by default.
type EnableableDefaultOff struct {
	// Enabled is a user-configurable value controlling enablement
	// Use IsEnabled to read this value.
	Enabled *bool `json:"enabled,omitempty"`
}

// IsEnabled tests whether this feature is enabled.
// EnableableDefaultOn features are considered disabled by default.
func (e EnableableDefaultOff) IsEnabled() bool {
	return e.Enabled != nil && *e.Enabled
}

type Prefix struct {
	EnableableDefaultOn
	Value string `json:"value,omitempty"`
}

type PipelinesDevelopment struct {
	EnableableDefaultOff
}

type TriggerPauseStatus struct {
	EnableableDefaultOff
}

type JobsMaxConcurrentRuns struct {
	EnableableDefaultOn
	Value int `json:"value,omitempty"`
}

type Tags struct {
	EnableableDefaultOn
	Tags map[string]string `json:"tags,omitempty"`
}

type Transformers struct {
	Prefix                Prefix                `json:"prefix,omitempty"`
	PipelinesDevelopment  PipelinesDevelopment  `json:"pipelines_development,omitempty"`
	TriggerPauseStatus    TriggerPauseStatus    `json:"trigger_pause_status,omitempty"`
	JobsMaxConcurrentRuns JobsMaxConcurrentRuns `json:"jobs_max_concurrent_runs,omitempty"`
	Tags                  Tags                  `json:"tags,omitempty"`
}
