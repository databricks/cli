package resources

// BaseResource is a struct that contains the base settings for a resource.
type BaseResource struct {
	URL            string         `json:"url,omitempty" bundle:"internal"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	Lifecycle      Lifecycle      `json:"lifecycle,omitempty"`
}

// GetLifecycle returns the lifecycle settings for the resource.
func (b *BaseResource) GetLifecycle() LifecycleConfig {
	return b.Lifecycle
}

// GetURL returns the resource's workspace URL, as set by InitializeURL.
func (b *BaseResource) GetURL() string {
	return b.URL
}
