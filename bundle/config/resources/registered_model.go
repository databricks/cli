package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type RegisteredModel struct {
	// This is a resource agnostic implementation of grants.
	// Implementation could be different based on the resource type.
	Grants []Grant `json:"grants,omitempty"`

	// This represents the id which is the full name of the model
	// (catalog_name.schema_name.model_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string

	// Path to config file where the resource is defined. All bundle resources
	// include this for interpolation purposes.
	paths.Paths

	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	*catalog.CreateRegisteredModelRequest
}
