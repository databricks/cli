package resources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ModelServingEndpoint struct {
	// dynamic value representation of the resource.
	DynamicValue dyn.Value

	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	*serving.CreateServingEndpoint

	// This represents the id (ie serving_endpoint_id) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// This is a resource agnostic implementation of permissions for ACLs.
	// Implementation could be different based on the resource type.
	Permissions []Permission `json:"permissions,omitempty"`

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}

func (s *ModelServingEndpoint) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ModelServingEndpoint) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *ModelServingEndpoint) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.ServingEndpoints.Get(ctx, serving.GetServingEndpointRequest{
		Name: id,
	})
	if err != nil {
		log.Debugf(ctx, "serving endpoint %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (s *ModelServingEndpoint) TerraformResourceName() string {
	return "databricks_model_serving"
}

func (s *ModelServingEndpoint) Validate() error {
	if s == nil || !s.DynamicValue.IsValid() {
		return fmt.Errorf("serving endpoint is not defined")
	}

	return nil
}
