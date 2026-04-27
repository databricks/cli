package resources

import (
	"net/url"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Connection is a UC foreign-catalog connection. Embeds the SDK's
// CreateConnection for full attribute coverage.
type Connection struct {
	catalog.CreateConnection

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets c.URL iff the connection has been deployed
// (ID is non-empty).
func (c *Connection) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	baseURL.Path = "explore/connections/" + c.Name
	c.URL = baseURL.String()
}

func (c *Connection) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c Connection) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}
