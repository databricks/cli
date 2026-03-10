package resources

// ILifecycle is implemented by Lifecycle and LifecycleWithStarted.
type ILifecycle interface {
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
	// Lifecycle setting to prevent the resource from being destroyed.
	PreventDestroy bool `json:"prevent_destroy,omitempty"`

	// If set to true, the resource will be deployed in started mode.
	// Supported only for apps, clusters, and sql_warehouses.
	Started *bool `json:"started,omitempty"`
}

// HasPreventDestroy returns true if prevent_destroy is set.
func (l LifecycleWithStarted) HasPreventDestroy() bool {
	return l.PreventDestroy
}
