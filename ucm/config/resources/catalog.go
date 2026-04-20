package resources

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
}
