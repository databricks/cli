package resources

import (
	"net/url"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Catalog is a UC catalog. Embeds the SDK's CreateCatalog so every API-level
// field is part of ucm's surface; ucm-specific concerns (tag validation, the
// post-deploy ID/URL pair) are sibling fields. Mirrors
// bundle/config/resources where Job/Schema/etc. embed their SDK request
// types.
type Catalog struct {
	catalog.CreateCatalog

	// Tags is a key/value map evaluated by ucm's tag-validation mutators.
	// Distinct from the SDK's `properties` (free-form) and `options` (managed
	// by the backend) — Tags is ucm-policy material.
	Tags map[string]string `json:"tags,omitempty"`

	// Schemas and Grants exist solely to keep convert.Normalize from
	// dropping nested-form keys during load — FlattenNestedResources
	// zeroes them before any other mutator runs. Always nil after load.
	Schemas map[string]*Schema `json:"schemas,omitempty"`
	Grants  map[string]*Grant  `json:"grants,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets c.URL iff the catalog has been deployed (ID is non-empty).
func (c *Catalog) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + c.Name
	c.URL = baseURL.String()
}

func (c *Catalog) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c Catalog) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}
