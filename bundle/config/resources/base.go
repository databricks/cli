package resources

// BaseResource is a struct that contains the base settings for a resource.
type BaseResource struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
	Lifecycle      Lifecycle      `json:"lifecycle,omitempty"`

	// DeployTargets optionally restricts the resource to a subset of targets.
	// When empty (the default) the resource is deployed to every target, matching
	// the historical behavior of a top-level resource. When set, the resource is
	// only deployed to a target whose name appears in the list; for any other
	// target it is dropped from the configuration before deployment. This lets a
	// single top-level definition be scoped to specific targets without nesting
	// the body under targets.<name>.resources. It is a bundle-only field and is
	// never emitted to the deployment engine.
	DeployTargets []string `json:"deploy_targets,omitempty"`
}

// GetLifecycle returns the lifecycle settings for the resource.
func (b *BaseResource) GetLifecycle() LifecycleConfig {
	return b.Lifecycle
}

// GetDeployTargets returns the targets the resource is restricted to, or an
// empty slice when the resource is not target-scoped.
func (b *BaseResource) GetDeployTargets() []string {
	return b.DeployTargets
}
