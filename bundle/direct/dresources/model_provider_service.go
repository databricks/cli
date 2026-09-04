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
		ForceSendFields:        nil,
	}, nil
}

// modelProviderServiceBody builds the ModelProviderService write payload from the
// bundle config. Only comment and config are client-settable; every other field
// is OUTPUT_ONLY (server-derived) and sent as its zero value.
func modelProviderServiceBody(config *resources.ModelProviderServiceConfig) catalog.ModelProviderService {
	return catalog.ModelProviderService{
		Comment:         config.Comment,
		Config:          config.Config,
		CreateTime:      nil,
		CreatedBy:       "",
		EffectiveOwner:  "",
		Etag:            "",
		MetastoreId:     "",
		Name:            "",
		Owner:           "",
		UpdateTime:      nil,
		UpdatedBy:       "",
		ForceSendFields: nil,
	}
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
		ModelProviderService:   modelProviderServiceBody(config),
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

// DoUpdate sends update_mask "*" on every update. name, parent and
// model_provider_service_id are immutable (provided_id_fields), and
// config.provider_type is immutable (recreate_on_changes), so the wildcard
// replaces every client-settable field (comment + a full config replace),
// matching the mask the Terraform provider generates.
//
// Etag is intentionally left empty here and in DoDelete: an empty etag means no
// If-Match precondition (last-write-wins), matching the Terraform provider,
// which also does not send etag. We deliberately do not do optimistic
// concurrency on these resources.
func (r *ResourceModelProviderService) DoUpdate(ctx context.Context, id string, config *resources.ModelProviderServiceConfig, _ *PlanEntry) (*resources.ModelProviderServiceConfig, error) {
	resp, err := r.client.AiGateway.UpdateModelProviderService(ctx, catalog.UpdateModelProviderServiceRequest{
		Etag:                 "",
		Name:                 modelProviderServiceNamePrefix + id,
		ModelProviderService: modelProviderServiceBody(config),
		UpdateMask:           fieldmask.FieldMask{Paths: []string{"*"}},
		ForceSendFields:      nil,
	})
	if err != nil {
		return nil, err
	}
	return responseToModelProviderServiceConfig(resp)
}

func (r *ResourceModelProviderService) DoDelete(ctx context.Context, id string, _ *resources.ModelProviderServiceConfig) error {
	return r.client.AiGateway.DeleteModelProviderService(ctx, catalog.DeleteModelProviderServiceRequest{
		Etag:            "",
		Name:            modelProviderServiceNamePrefix + id,
		ForceSendFields: nil,
	})
}
