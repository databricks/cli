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

func (*ModelServingEndpoint) RunAsSupport() RunAsSupport {
	return RunAsRejected
}

func (*ModelServingEndpoint) Approval() Approval {
	return Approval{}
}

func (s *ModelServingEndpoint) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model_serving_endpoint",
		PluralName:    "model_serving_endpoints",
		SingularTitle: "Model Serving Endpoint",
		PluralTitle:   "Model Serving Endpoints",
	}
}

func (*ModelServingEndpoint) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_VIEW",
			CanRun:    "CAN_QUERY",
		},
		RequestObjectType:  "serving-endpoints",
		ResponseObjectType: "serving-endpoint",
		ObjectIDField:      "endpoint_id", // internal numeric ID, not the name used in CRUD APIs
	}
}

func (*ModelServingEndpoint) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "ml/endpoints/%s"}, true
}

func (s *ModelServingEndpoint) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *ModelServingEndpoint) GetName() string {
	return s.Name
}

func (s *ModelServingEndpoint) UCNames() []UCName {
	var names []UCName
	if s.AiGateway != nil && s.AiGateway.InferenceTableConfig != nil {
		itc := s.AiGateway.InferenceTableConfig
		names = append(names, ucCatalogAndSchemas(&itc.CatalogName, &itc.SchemaName))
	}
	// AutoCaptureConfig is deprecated but still in use.
	if s.Config != nil && s.Config.AutoCaptureConfig != nil {
		acc := s.Config.AutoCaptureConfig
		names = append(names, ucCatalogAndSchemas(&acc.CatalogName, &acc.SchemaName))
	}
	return names
}

func (s *ModelServingEndpoint) ApplyPresets(p *PresetSettings) {
	s.Name = normalizePrefix(p.NamePrefix) + s.Name

	// As of 2024-06, model serving endpoints don't yet support tags
}
