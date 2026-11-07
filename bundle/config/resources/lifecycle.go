package resources

// LifecycleConfig is implemented by Lifecycle and LifecycleWithStarted.
type LifecycleConfig interface {
	HasPreventDestroy() bool
	GetDependsOn() []DependsOn
	GetBind() *Bind
}

// Lifecycle contains base lifecycle settings supported by all resources.
type Lifecycle struct {
	// Lifecycle setting to prevent the resource from being destroyed.
	PreventDestroy bool `json:"prevent_destroy,omitempty"`

	// ForceSendFields records which fields were explicitly set so that zero-value
	// overrides can replace non-zero base values (e.g., prevent_destroy: false in a target
	// can override prevent_destroy: true in the base).
	// This field is not serialized to JSON.
	ForceSendFields []string `json:"-"`

	// Resources that must be deployed before this resource and deleted after it.
	DependsOn []DependsOn `json:"depends_on,omitempty"`

	// Existing workspace resource to bring under bundle management instead of creating one.
	Bind *Bind `json:"bind,omitempty"`
}

// DependsOn is a single lifecycle.depends_on entry.
type DependsOn struct {
	// Key of the resource to depend on: "foo" or "jobs.foo".
	Resource string `json:"resource"`
}

// Bind is the lifecycle.bind block: the deploy adopts the workspace resource with this
// ID into the deployment state instead of creating a new one. Nothing is recorded in
// state about the adoption; the block guards the resource against being recreated for
// as long as it is present in the configuration.
type Bind struct {
	// ID of the existing workspace resource, as used by its API.
	ID string `json:"id"`
}

// HasPreventDestroy returns true if prevent_destroy is set.
func (l Lifecycle) HasPreventDestroy() bool {
	return l.PreventDestroy
}

// GetDependsOn returns the lifecycle.depends_on entries.
func (l Lifecycle) GetDependsOn() []DependsOn {
	return l.DependsOn
}

// GetBind returns the lifecycle.bind block, or nil when the resource is not bound.
func (l Lifecycle) GetBind() *Bind {
	return l.Bind
}

// LifecycleWithStarted contains lifecycle settings for resources that support lifecycle.started.
// It is used by apps, clusters, and sql_warehouses.
type LifecycleWithStarted struct {
	Lifecycle

	// If set to true, the resource will be deployed in started mode.
	// Supported only for apps, clusters, and sql_warehouses.
	Started *bool `json:"started,omitempty"`
}
