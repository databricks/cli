package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	marshal "github.com/databricks/databricks-sdk-go/json"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ModelServingEndpoint struct {
	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	*serving.CreateServingEndpoint

	// This represents the id (ie serving_endpoint_id) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string

	// Local path where the bundle is defined. All bundle resources include
	// this for interpolation purposes.
	paths.Paths

	// This is a resource agnostic implementation of permissions for ACLs.
	// Implementation could be different based on the resource type.
	Permissions []Permission `json:"permissions,omitempty"`
}

func (s *ModelServingEndpoint) UnmarshalJSON(b []byte) error {
	type C ModelServingEndpoint
	return marshal.Unmarshal(b, (*C)(s))
}

func (s ModelServingEndpoint) MarshalJSON() ([]byte, error) {
	type C ModelServingEndpoint
	return marshal.Marshal((C)(s))
}
