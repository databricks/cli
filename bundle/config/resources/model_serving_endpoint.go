package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ModelServingEndpoint struct {
	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	*serving.CreateServingEndpoint

	// This represents the id (ie serving_endpoint_id) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// Path to config file where the resource is defined. All bundle resources
	// include this for interpolation purposes.
	paths.Paths

	// This is a resource agnostic implementation of permissions for ACLs.
	// Implementation could be different based on the resource type.
	Permissions []Permission `json:"permissions,omitempty"`

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"readonly"`
}

func (s *ModelServingEndpoint) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ModelServingEndpoint) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
