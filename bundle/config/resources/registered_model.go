package resources

import (
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type RegisteredModel struct {
	// This is a resource agnostic implementation of grants.
	// Implementation could be different based on the resource type.
	Grants []Grant `json:"grants,omitempty"`

	// This represents the id which is the full name of the model
	// (catalog_name.schema_name.model_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	*catalog.CreateRegisteredModelRequest

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}

func (s *RegisteredModel) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s RegisteredModel) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
