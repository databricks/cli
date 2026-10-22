package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ModelServingEndpoint struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`

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

var modelServingEndpointDescription = ResourceDescription{
	SingularName:  "model_serving_endpoint",
	PluralName:    "model_serving_endpoints",
	SingularTitle: "Model Serving Endpoint",
	PluralTitle:   "Model Serving Endpoints",
	Permissions: PermissionsDescription{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_VIEW",
			CanRun:    "CAN_QUERY",
		},
		RequestObjectType:  "serving-endpoints",
		ResponseObjectType: "serving-endpoint",
		ObjectIDField:      "endpoint_id", // internal numeric ID, not the name used in CRUD APIs
	},
}

func (s *ModelServingEndpoint) ResourceDescription() ResourceDescription {
	return modelServingEndpointDescription
}

func (s *ModelServingEndpoint) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "model_serving_endpoints", s.ID)
}

func (s *ModelServingEndpoint) GetName() string {
	return s.Name
}
