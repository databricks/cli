package resources

// LifecycleConfig is implemented by Lifecycle and LifecycleWithStarted.
type LifecycleConfig interface {
	HasPreventDestroy() bool
}

// Lifecycle contains base lifecycle settings supported by all resources.
type Lifecycle struct {
	// Lifecycle setting to prevent the resource from being destroyed.
	PreventDestroy bool `json:"prevent_destroy,omitempty"`
}

// HasPreventDestroy returns true if prevent_destroy is set.
func (l Lifecycle) HasPreventDestroy() bool {
	return l.PreventDestroy
}

// LifecycleWithStarted contains lifecycle settings for resources that support lifecycle.started.
// It is used by apps, clusters, and sql_warehouses.
type LifecycleWithStarted struct {
	Lifecycle

	// If set to true, the resource will be deployed in started mode.
	// Supported only for apps, clusters, and sql_warehouses.
	Started *bool `json:"started,omitempty"`
}

// JobRunLifecycle adds run-fire triggers; other resources keep base Lifecycle only.
type JobRunLifecycle struct {
	Lifecycle

	// Without triggers, the run re-fires only when its own config changes.
	Triggers []JobRunTrigger `json:"triggers,omitempty"`
}

// JobRunTrigger is one lifecycle.triggers entry. Exactly one field must be set.
type JobRunTrigger struct {
	OnBundleDeploy *bool  `json:"on_bundle_deploy,omitempty"`
	OnFileChange   string `json:"on_file_change,omitempty"`
	OnValueChange  string `json:"on_value_change,omitempty"`
}

// FieldCount returns how many trigger modes are set on this entry.
func (t JobRunTrigger) FieldCount() int {
	n := 0
	if t.OnBundleDeploy != nil {
		n++
	}
	if t.OnFileChange != "" {
		n++
	}
	if t.OnValueChange != "" {
		n++
	}
	return n
}
