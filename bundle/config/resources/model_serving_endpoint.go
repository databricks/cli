package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ModelServingEndpoint struct {
	BaseResource

	// This represents the input args for terraform, and will get converted
	// to a HCL representation for CRUD
	serving.CreateServingEndpoint

	// This is a resource agnostic implementation of permissions for ACLs.
	// Implementation could be different based on the resource type.
	Permissions []ModelServingEndpointPermission `json:"permissions,omitempty"`
}

func (s *ModelServingEndpoint) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ModelServingEndpoint) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *ModelServingEndpoint) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.ServingEndpoints.Get(ctx, serving.GetServingEndpointRequest{
		Name: name,
	})
	if err != nil {
		log.Debugf(ctx, "serving endpoint %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (s *ModelServingEndpoint) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model_serving_endpoint",
		PluralName:    "model_serving_endpoints",
		SingularTitle: "Model Serving Endpoint",
		PluralTitle:   "Model Serving Endpoints",
	}
}

func (s *ModelServingEndpoint) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "ml/endpoints/" + s.ID
	s.URL = baseURL.String()
}

func (s *ModelServingEndpoint) GetName() string {
	return s.Name
}

func (s *ModelServingEndpoint) GetURL() string {
	return s.URL
}
