package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/utils"
	sdkmodelserving "github.com/databricks/sdk-go/modelserving/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type ModelServingEndpoint struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`

	modelserving.CreateInferenceEndpointRequest

	// This is a resource agnostic implementation of permissions for ACLs.
	// Implementation could be different based on the resource type.
	Permissions []ModelServingEndpointPermission `json:"permissions,omitempty"`
}

func (s *ModelServingEndpoint) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.ServingEndpoints.GetInferenceEndpoint(ctx, sdkmodelserving.GetInferenceEndpointRequest{
		Name: new(name),
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
	s.URL = workspaceurls.ResourceURL(baseURL, "model_serving_endpoints", s.ID)
}

func (s *ModelServingEndpoint) GetName() string {
	return utils.Deref(s.Name)
}
