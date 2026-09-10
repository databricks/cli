package resources

// BaseResource is a struct that contains the base settings for a resource.
type BaseResource struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
	Lifecycle      Lifecycle      `json:"lifecycle,omitempty"`
}

// GetLifecycle returns the lifecycle settings for the resource.
func (b *BaseResource) GetLifecycle() LifecycleConfig {
	return b.Lifecycle
}

// GetURL returns the resource's workspace URL and true. Resource types whose
// IDs don't map to a web UI page override this to return ("", false).
func (b *BaseResource) GetURL() (string, bool) {
	return b.URL, true
}
