package resources

// UCName points at fields of a resource that name a Unity Catalog catalog or
// schema. Either Catalog or Compound is set.
type UCName struct {
	// Catalog holds a catalog name, and each of Schemas the name of a schema in
	// that catalog.
	Catalog *string
	Schemas []*string

	// Compound holds Prefix followed by Parts dot-separated components, the first
	// being a catalog name and the second the name of a schema in that catalog.
	Compound *string
	Prefix   string
	Parts    int
}

// UCNameDeclarer is implemented by a resource that names Unity Catalog catalogs
// or schemas. A name that matches a catalog or schema defined in the same bundle
// is rewritten into a reference to that resource, so that the deployment order
// follows the dependency.
//
// Unlike a static declaration, this is per-instance behaviour: UCNames is called on
// each configured resource and returns pointers into it, so that the names can be
// rewritten in place.
type UCNameDeclarer interface {
	// UCNames returns pointers to the fields of the receiver that name a catalog or
	// schema.
	UCNames() []UCName
}

// UCCatalogResource is implemented by a resource that defines a Unity Catalog
// catalog. A reference to it points at its "name" field.
type UCCatalogResource interface {
	UCCatalogName() string
}

// UCSchemaResource is implemented by a resource that defines a Unity Catalog
// schema. A reference to it points at its "name" field.
type UCSchemaResource interface {
	UCSchemaName() (catalogName, schemaName string)
}

// ucCatalogAndSchemas declares a field holding a catalog name and fields holding
// names of schemas in that catalog.
func ucCatalogAndSchemas(catalog *string, schemas ...*string) UCName {
	return UCName{Catalog: catalog, Schemas: schemas}
}

// ucSchemaFullName declares a field holding "catalog.schema".
func ucSchemaFullName(name *string) UCName {
	return UCName{Compound: name, Parts: 2}
}

// ucThreePartName declares a field holding "catalog.schema.name".
func ucThreePartName(name *string) UCName {
	return UCName{Compound: name, Parts: 3}
}

// ucSchemaParent declares a field holding "schemas/catalog.schema", the parent
// of a securable that lives in a schema.
func ucSchemaParent(parent *string) UCName {
	return UCName{Compound: parent, Prefix: "schemas/", Parts: 2}
}
