package resources

import "net/url"

// Schema is a UC schema (a.k.a. database) nested inside a catalog.
type Schema struct {
	// Name of the schema. Required.
	Name string `json:"name"`

	// Catalog is the name of the parent catalog. Required in flat form;
	// injected by FlattenNestedResources when declared nested under a catalog.
	// In M1 this becomes interpolatable via ${resources.catalogs.X.name}.
	Catalog string `json:"catalog,omitempty"`

	// Comment is a human-readable description.
	Comment string `json:"comment,omitempty"`

	// Tags is a key/value map evaluated by ucm's tag-validation mutators.
	Tags map[string]string `json:"tags,omitempty"`

	// TagInherit controls whether parent-catalog tags merge into this schema's
	// tags (schema-key wins). nil means inherit (the default). Set false to
	// opt out.
	TagInherit *bool `json:"tag_inherit,omitempty"`

	// Grants nested under this schema. FlattenNestedResources moves them to
	// Root.Resources.Grants with securable={type:schema, name:<this>} injected.
	// Always nil after load.
	Grants map[string]*Grant `json:"grants,omitempty"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

func (s *Schema) InitializeURL(baseURL url.URL) {
	if s.Catalog == "" || s.Name == "" {
		return
	}
	baseURL.Path = "explore/data/" + s.Catalog + "/" + s.Name
	s.URL = baseURL.String()
}
