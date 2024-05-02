package resources

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Schema struct {
	// List of grants to apply on this schema.
	Grants []Grant `json:"grants,omitempty"`

	// This represents the id which is the full name of the schema
	// (catalog_name.schema_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	// TODO: verify the accuracy of this comment, it just might be the schema name
	ID string `json:"id,omitempty" bundle:"readonly"`

	*catalog.CreateSchema

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}
