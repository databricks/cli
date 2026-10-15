package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	aigateway "github.com/databricks/cli/libs/sdktypes/aigateway/v1"
	"github.com/databricks/cli/libs/utils"
	sdkaigateway "github.com/databricks/sdk-go/aigateway/v1"
)

// AI Gateway MCP service.
// API: https://docs.databricks.com/api/workspace/aigateway
// Terraform: databricks_ai_gateway_mcp_service
//
// Mirrors ResourceModelService: the remote type returned by DoRead is the same
// bundle-local resources.McpServiceConfig used for state (so RemapState is not
// needed), and DoRead reconstructs the create-time identity (parent +
// mcp_service_id) from the server-derived resource name.
const mcpServiceNamePrefix = "mcp-services/"

type ResourceMcpService struct {
	client *sdkclients.Clients
}

func (*ResourceMcpService) New(client *sdkclients.Clients) *ResourceMcpService {
	return &ResourceMcpService{client: client}
}

func (*ResourceMcpService) PrepareState(input *resources.McpService) *resources.McpServiceConfig {
	return &input.McpServiceConfig
}

// mcpServiceIdentityFromName reconstructs the create-time parent and leaf id
// from the server-derived resource name
// `mcp-services/{catalog}.{schema}.{mcp_service}`.
func mcpServiceIdentityFromName(name string) (parent, mcpServiceId string, err error) {
	rest, ok := strings.CutPrefix(name, mcpServiceNamePrefix)
	if !ok {
		return "", "", fmt.Errorf("unexpected mcp service name %q (want mcp-services/{catalog}.{schema}.{mcp_service})", name)
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("unexpected mcp service name %q (want three dot-separated components)", name)
	}
	return "schemas/" + parts[0] + "." + parts[1], parts[2], nil
}

func responseToMcpServiceConfig(resp *sdkaigateway.McpService) (*resources.McpServiceConfig, error) {
	ms, err := aigateway.McpServiceFromSDK(resp)
	if err != nil {
		return nil, err
	}
	parent, id, err := mcpServiceIdentityFromName(utils.Deref(ms.Name))
	if err != nil {
		return nil, err
	}
	return &resources.McpServiceConfig{
		Parent:       parent,
		McpServiceId: id,
		Comment:      utils.Deref(ms.Comment),
		Config:       ms.Config,
	}, nil
}

// mcpServiceBody builds the McpService write payload from the bundle config.
// Only comment and config are client-settable; every other field is OUTPUT_ONLY
// (server-derived) and not sent. name addresses the resource on update.
func mcpServiceBody(config *resources.McpServiceConfig, name *string) *aigateway.McpService {
	return &aigateway.McpService{
		Name:           name,
		Comment:        nonEmpty(config.Comment),
		Config:         config.Config,
		CreateTime:     nil, // Not set by DABs
		CreatedBy:      nil, // Not set by DABs
		EffectiveOwner: nil, // Not set by DABs
		Etag:           nil, // Not set by DABs
		MetastoreId:    nil, // Not set by DABs
		UpdateTime:     nil, // Not set by DABs
		UpdatedBy:      nil, // Not set by DABs
	}
}

func (r *ResourceMcpService) DoRead(ctx context.Context, id string) (*resources.McpServiceConfig, error) {
	ms, err := r.client.AiGateway.GetMcpService(ctx, sdkaigateway.GetMcpServiceRequest{Name: new(mcpServiceNamePrefix + id)})
	if err != nil {
		return nil, err
	}
	return responseToMcpServiceConfig(ms)
}

func (r *ResourceMcpService) DoCreate(ctx context.Context, config *resources.McpServiceConfig) (string, *resources.McpServiceConfig, error) {
	request, err := aigateway.CreateMcpServiceRequestToSDK(&aigateway.CreateMcpServiceRequest{
		Parent:       new(config.Parent),
		McpServiceId: new(config.McpServiceId),
		McpService:   mcpServiceBody(config, nil),
	})
	if err != nil {
		return "", nil, err
	}
	resp, err := r.client.AiGateway.CreateMcpService(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	state, err := responseToMcpServiceConfig(resp)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimPrefix(utils.Deref(resp.Name), mcpServiceNamePrefix), state, nil
}

// DoUpdate sends update_mask "*" on every update. name, parent and
// mcp_service_id are immutable (provided_id_fields in resources.yml), so the
// wildcard replaces every client-settable field (comment + a full config
// replace), matching the mask the Terraform provider generates.
//
// Etag is intentionally left empty here and in DoDelete: an empty etag means no
// If-Match precondition (last-write-wins), matching the Terraform provider,
// which also does not send etag. We deliberately do not do optimistic
// concurrency on these resources.
func (r *ResourceMcpService) DoUpdate(ctx context.Context, id string, config *resources.McpServiceConfig, _ *PlanEntry) (*resources.McpServiceConfig, error) {
	request, err := aigateway.UpdateMcpServiceRequestToSDK(&aigateway.UpdateMcpServiceRequest{
		McpService: mcpServiceBody(config, new(mcpServiceNamePrefix+id)),
		UpdateMask: new("*"),
		Etag:       nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	resp, err := r.client.AiGateway.UpdateMcpService(ctx, *request)
	if err != nil {
		return nil, err
	}
	return responseToMcpServiceConfig(resp)
}

func (r *ResourceMcpService) DoDelete(ctx context.Context, id string, _ *resources.McpServiceConfig) error {
	return r.client.AiGateway.DeleteMcpService(ctx, sdkaigateway.DeleteMcpServiceRequest{
		Name: new(mcpServiceNamePrefix + id),
		Etag: nil, // Not set by DABs
	})
}
