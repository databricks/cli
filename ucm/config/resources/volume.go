package resources

import (
	"net/url"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Volume is a UC volume (managed or external). Embeds the SDK's
// CreateVolumeRequestContent for full attribute coverage; ucm-specific
// concerns (post-deploy ID/URL pair) are sibling fields.
type Volume struct {
	catalog.CreateVolumeRequestContent

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets v.URL iff the volume has been deployed (ID is non-empty).
func (v *Volume) InitializeURL(baseURL url.URL) {
	if v.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + v.CatalogName + "/" + v.SchemaName + "/" + v.Name
	v.URL = baseURL.String()
}

func (v *Volume) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, v)
}

func (v Volume) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(v)
}
