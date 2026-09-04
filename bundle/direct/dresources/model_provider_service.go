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

// AI Gateway model provider service.
// API: https://docs.databricks.com/api/workspace/aigateway
// Terraform: databricks_ai_gateway_model_provider_service
//
// Mirrors ResourceModelService: the remote type returned by DoRead is the same
// bundle-local resources.ModelProviderServiceConfig used for state (so
// RemapState is not needed), and DoRead reconstructs the create-time identity
// (parent + model_provider_service_id) from the server-derived resource name.
const modelProviderServiceNamePrefix = "model-provider-services/"

type ResourceModelProviderService struct {
	client *databricks.WorkspaceClient
}

func (*ResourceModelProviderService) New(client *databricks.WorkspaceClient) *ResourceModelProviderService {
	return &ResourceModelProviderService{client: client}
}

func (*ResourceModelProviderService) PrepareState(input *resources.ModelProviderService) *resources.ModelProviderServiceConfig {
	return &input.ModelProviderServiceConfig
}

// modelProviderServiceIdentityFromName reconstructs the create-time parent and
// leaf id from the server-derived resource name
// `model-provider-services/{catalog}.{schema}.{model_provider_service}`.
func modelProviderServiceIdentityFromName(name string) (parent, modelProviderServiceId string, err error) {
	rest, ok := strings.CutPrefix(name, modelProviderServiceNamePrefix)
	if !ok {
		return "", "", fmt.Errorf("unexpected model provider service name %q (want model-provider-services/{catalog}.{schema}.{model_provider_service})", name)
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("unexpected model provider service name %q (want three dot-separated components)", name)
	}
	return "schemas/" + parts[0] + "." + parts[1], parts[2], nil
}

func responseToModelProviderServiceConfig(ms *catalog.ModelProviderService) (*resources.ModelProviderServiceConfig, error) {
	parent, id, err := modelProviderServiceIdentityFromName(ms.Name)
	if err != nil {
		return nil, err
	}
	return &resources.ModelProviderServiceConfig{
		Parent:                 parent,
		ModelProviderServiceId: id,
		Comment:                ms.Comment,
		Config:                 ms.Config,
	}, nil
}

func (r *ResourceModelProviderService) DoRead(ctx context.Context, id string) (*resources.ModelProviderServiceConfig, error) {
	ms, err := r.client.AiGateway.GetModelProviderService(ctx, catalog.GetModelProviderServiceRequest{Name: modelProviderServiceNamePrefix + id})
	if err != nil {
		return nil, err
	}
	return responseToModelProviderServiceConfig(ms)
}

func (r *ResourceModelProviderService) DoCreate(ctx context.Context, config *resources.ModelProviderServiceConfig) (string, *resources.ModelProviderServiceConfig, error) {
	resp, err := r.client.AiGateway.CreateModelProviderService(ctx, catalog.CreateModelProviderServiceRequest{
		Parent:                 config.Parent,
		ModelProviderServiceId: config.ModelProviderServiceId,
		ModelProviderService: catalog.ModelProviderService{
			Comment: config.Comment,
			Config:  config.Config,
		},
	})
	if err != nil {
		return "", nil, err
	}
	state, err := responseToModelProviderServiceConfig(resp)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimPrefix(resp.Name, modelProviderServiceNamePrefix), state, nil
}

// modelProviderServiceUpdateMask lists the mutable fields sent on every update.
// name, parent and model_provider_service_id are immutable (provided_id_fields),
// and config.provider_type is immutable (recreate_on_changes). The API rejects
// wildcard masks, so each path is explicit.
var modelProviderServiceUpdateMask = []string{"comment", "config"}

func (r *ResourceModelProviderService) DoUpdate(ctx context.Context, id string, config *resources.ModelProviderServiceConfig, _ *PlanEntry) (*resources.ModelProviderServiceConfig, error) {
	resp, err := r.client.AiGateway.UpdateModelProviderService(ctx, catalog.UpdateModelProviderServiceRequest{
		Name: modelProviderServiceNamePrefix + id,
		ModelProviderService: catalog.ModelProviderService{
			Comment: config.Comment,
			Config:  config.Config,
		},
		UpdateMask: fieldmask.FieldMask{Paths: modelProviderServiceUpdateMask},
	})
	if err != nil {
		return nil, err
	}
	return responseToModelProviderServiceConfig(resp)
}

func (r *ResourceModelProviderService) DoDelete(ctx context.Context, id string, _ *resources.ModelProviderServiceConfig) error {
	return r.client.AiGateway.DeleteModelProviderService(ctx, catalog.DeleteModelProviderServiceRequest{Name: modelProviderServiceNamePrefix + id})
}
