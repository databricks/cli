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

// JobRunLifecycle extends Lifecycle with run-fire triggers.
type JobRunLifecycle struct {
	Lifecycle

	// Triggers that cause the run to re-fire (in addition to config changes).
	Triggers []JobRunTrigger `json:"triggers,omitempty"`
}

// JobRunTrigger is one lifecycle.triggers entry.
type JobRunTrigger struct {
	OnBundleDeploy *bool   `json:"on_bundle_deploy,omitempty"`
	OnFileChange   *string `json:"on_file_change,omitempty"`  // path or glob relative to the defining YAML file; must resolve under the sync root
	OnValueChange  *string `json:"on_value_change,omitempty"` // interpolated expression; re-fire when its resolved fingerprint changes
}

// ArmedCount returns the number of trigger fields set on this entry.
func (t JobRunTrigger) ArmedCount() int {
	n := 0
	if t.OnBundleDeploy != nil {
		n++
	}
	if t.OnFileChange != nil {
		n++
	}
	if t.OnValueChange != nil {
		n++
	}
	return n
}
