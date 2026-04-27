package resources

import (
	"net/url"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ExternalLocation is a UC external location. Embeds the SDK's
// CreateExternalLocation for full attribute coverage.
//
// SDK's `Url` field (lowercase, the cloud storage path) is distinct from
// ucm's `URL` field (uppercase, the workspace console URL set by
// initialize_urls).
type ExternalLocation struct {
	catalog.CreateExternalLocation

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"workspace_url,omitempty" ucm:"readonly"`
}

// InitializeURL sets e.URL iff the external location has been deployed
// (ID is non-empty).
func (e *ExternalLocation) InitializeURL(baseURL url.URL) {
	if e.ID == "" {
		return
	}
	baseURL.Path = "explore/external-locations/" + e.Name
	e.URL = baseURL.String()
}

func (e *ExternalLocation) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, e)
}

func (e ExternalLocation) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(e)
}
