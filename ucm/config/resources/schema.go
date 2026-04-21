package resources

// Schema is a UC schema (a.k.a. database) nested inside a catalog.
type Schema struct {
	// Name of the schema. Required.
	Name string `json:"name"`

	// Catalog is the name of the parent catalog. Required.
	// In M1 this becomes interpolatable via ${resources.catalogs.X.name}.
	Catalog string `json:"catalog"`

	// Comment is a human-readable description.
	Comment string `json:"comment,omitempty"`

	// Tags is a key/value map evaluated by ucm's tag-validation mutators.
	Tags map[string]string `json:"tags,omitempty"`
}
