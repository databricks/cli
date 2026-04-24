package resources

import "net/url"

// Catalog is a UC catalog. M0 scope: name, comment, storage_root, tags.
// Additional fields (owner, properties, isolation_mode, etc.) will land in M1.
type Catalog struct {
	// Name of the catalog in Unity Catalog. Required.
	Name string `json:"name"`

	// Comment is a human-readable description.
	Comment string `json:"comment,omitempty"`

	// StorageRoot is the cloud storage URL backing the catalog (optional for
	// managed catalogs without an explicit storage root).
	StorageRoot string `json:"storage_root,omitempty"`

	// Tags is a key/value map evaluated by ucm's tag-validation mutators.
	Tags map[string]string `json:"tags,omitempty"`

	// Schemas and Grants are nested-form conveniences: the FlattenNestedResources
	// mutator moves them to Root.Resources.{Schemas,Grants} (injecting parent
	// references) before any other mutator runs. Always nil after load.
	Schemas map[string]*Schema `json:"schemas,omitempty"`
	Grants  map[string]*Grant  `json:"grants,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets c.URL iff the catalog has been deployed (ID is non-empty).
// Mirrors bundle/config/resources.Job.InitializeURL's ID-gated pattern so
// `ucm summary` only prints URLs that actually resolve.
func (c *Catalog) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + c.Name
	c.URL = baseURL.String()
}
