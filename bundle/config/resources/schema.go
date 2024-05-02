package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Schema struct {
	// List of grants to apply on this schema.
	Grants []Grant `json:"grants,omitempty"`

	// This represents the id which is the full name of the schema
	// (catalog_name.schema_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	// TODO: verify the accuracy of this comment
	ID string `json:"id,omitempty" bundle:"readonly"`

	// Path to config file where the resource is defined. All bundle resources
	// include this for interpolation purposes.
	paths.Paths

	*catalog.CreateSchema

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}
