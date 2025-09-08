package resources

// BaseResource is a struct that contains the base settings for a resource.
type BaseResource struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
}
