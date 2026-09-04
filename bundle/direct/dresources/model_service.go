package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// AI Gateway model service.
// API: https://docs.databricks.com/api/workspace/aigateway
// Terraform: databricks_ai_gateway_model_service
//
// The remote type returned by DoRead is the same bundle-local
// resources.ModelServiceConfig used for state, so RemapState is not needed.
// DoRead reconstructs the create-time identity (parent + model_service_id) from
// the server-derived resource name so those fields participate in normal drift
// detection rather than being suppressed as missing-in-remote.
// modelServiceNamePrefix is the fixed prefix of the resource name
// (model-services/{catalog}.{schema}.{model_service}). The engine tracks the id
// as the bare {catalog}.{schema}.{model_service} portion and this prefix is
// re-added when addressing the resource through the SDK.
const modelServiceNamePrefix = "model-services/"

type ResourceModelService struct {
	client *databricks.WorkspaceClient
}

func (*ResourceModelService) New(client *databricks.WorkspaceClient) *ResourceModelService {
	return &ResourceModelService{client: client}
}

func (*ResourceModelService) PrepareState(input *resources.ModelService) *resources.ModelServiceConfig {
	return &input.ModelServiceConfig
}

// modelServiceIdentityFromName reconstructs the create-time parent and leaf id
// from the server-derived resource name
// `model-services/{catalog}.{schema}.{model_service}`.
func modelServiceIdentityFromName(name string) (parent, modelServiceId string, err error) {
	rest, ok := strings.CutPrefix(name, "model-services/")
	if !ok {
		return "", "", fmt.Errorf("unexpected model service name %q (want model-services/{catalog}.{schema}.{model_service})", name)
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("unexpected model service name %q (want three dot-separated components)", name)
	}
	return "schemas/" + parts[0] + "." + parts[1], parts[2], nil
}

func responseToModelServiceConfig(ms *catalog.ModelService) (*resources.ModelServiceConfig, error) {
	parent, id, err := modelServiceIdentityFromName(ms.Name)
	if err != nil {
		return nil, err
	}
	return &resources.ModelServiceConfig{
		Parent:         parent,
		ModelServiceId: id,
		Comment:        ms.Comment,
		Config:         ms.Config,
	}, nil
}

func (r *ResourceModelService) DoRead(ctx context.Context, id string) (*resources.ModelServiceConfig, error) {
	ms, err := r.client.AiGateway.GetModelService(ctx, catalog.GetModelServiceRequest{Name: modelServiceNamePrefix + id})
	if err != nil {
		return nil, err
	}
	return responseToModelServiceConfig(ms)
}

func (r *ResourceModelService) DoCreate(ctx context.Context, config *resources.ModelServiceConfig) (string, *resources.ModelServiceConfig, error) {
	resp, err := r.client.AiGateway.CreateModelService(ctx, catalog.CreateModelServiceRequest{
		Parent:         config.Parent,
		ModelServiceId: config.ModelServiceId,
		ModelService: catalog.ModelService{
			Comment: config.Comment,
			Config:  config.Config,
		},
	})
	if err != nil {
		return "", nil, err
	}
	state, err := responseToModelServiceConfig(resp)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimPrefix(resp.Name, modelServiceNamePrefix), state, nil
}

// modelServiceUpdateMask lists the mutable fields sent on every update. name,
// parent and model_service_id are immutable (recreate_on_changes in
// resources.yml). The API rejects wildcard masks, so each path is explicit.
var modelServiceUpdateMask = []string{"comment", "config"}

func (r *ResourceModelService) DoUpdate(ctx context.Context, id string, config *resources.ModelServiceConfig, _ *PlanEntry) (*resources.ModelServiceConfig, error) {
	resp, err := r.client.AiGateway.UpdateModelService(ctx, catalog.UpdateModelServiceRequest{
		Name: modelServiceNamePrefix + id,
		ModelService: catalog.ModelService{
			Comment: config.Comment,
			Config:  config.Config,
		},
		UpdateMask: fieldmask.FieldMask{Paths: modelServiceUpdateMask},
	})
	if err != nil {
		return nil, err
	}
	return responseToModelServiceConfig(resp)
}

func (r *ResourceModelService) DoDelete(ctx context.Context, id string, _ *resources.ModelServiceConfig) error {
	return r.client.AiGateway.DeleteModelService(ctx, catalog.DeleteModelServiceRequest{Name: modelServiceNamePrefix + id})
}
