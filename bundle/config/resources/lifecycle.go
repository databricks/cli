package resources

// Lifecycle is a struct that contains the lifecycle settings for a resource.
// It controls the behavior of the resource when it is deployed or destroyed.
type Lifecycle struct {
	// Lifecycle setting to prevent the resource from being destroyed.
	PreventDestroy bool `json:"prevent_destroy,omitempty"`
}
