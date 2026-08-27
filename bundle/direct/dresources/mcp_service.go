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
	client *databricks.WorkspaceClient
}

func (*ResourceMcpService) New(client *databricks.WorkspaceClient) *ResourceMcpService {
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

func responseToMcpServiceConfig(ms *catalog.McpService) (*resources.McpServiceConfig, error) {
	parent, id, err := mcpServiceIdentityFromName(ms.Name)
	if err != nil {
		return nil, err
	}
	return &resources.McpServiceConfig{
		Parent:       parent,
		McpServiceId: id,
		Comment:      ms.Comment,
		Config:       ms.Config,
	}, nil
}

func (r *ResourceMcpService) DoRead(ctx context.Context, id string) (*resources.McpServiceConfig, error) {
	ms, err := r.client.AiGateway.GetMcpService(ctx, catalog.GetMcpServiceRequest{Name: mcpServiceNamePrefix + id})
	if err != nil {
		return nil, err
	}
	return responseToMcpServiceConfig(ms)
}

func (r *ResourceMcpService) DoCreate(ctx context.Context, config *resources.McpServiceConfig) (string, *resources.McpServiceConfig, error) {
	resp, err := r.client.AiGateway.CreateMcpService(ctx, catalog.CreateMcpServiceRequest{
		Parent:       config.Parent,
		McpServiceId: config.McpServiceId,
		McpService: catalog.McpService{
			Comment: config.Comment,
			Config:  config.Config,
		},
	})
	if err != nil {
		return "", nil, err
	}
	state, err := responseToMcpServiceConfig(resp)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimPrefix(resp.Name, mcpServiceNamePrefix), state, nil
}

// mcpServiceUpdateMask lists the mutable fields sent on every update. name,
// parent and mcp_service_id are immutable (provided_id_fields in resources.yml).
// The API rejects wildcard masks, so each path is explicit.
var mcpServiceUpdateMask = []string{"comment", "config"}

func (r *ResourceMcpService) DoUpdate(ctx context.Context, id string, config *resources.McpServiceConfig, _ *PlanEntry) (*resources.McpServiceConfig, error) {
	resp, err := r.client.AiGateway.UpdateMcpService(ctx, catalog.UpdateMcpServiceRequest{
		Name: mcpServiceNamePrefix + id,
		McpService: catalog.McpService{
			Comment: config.Comment,
			Config:  config.Config,
		},
		UpdateMask: fieldmask.FieldMask{Paths: mcpServiceUpdateMask},
	})
	if err != nil {
		return nil, err
	}
	return responseToMcpServiceConfig(resp)
}

func (r *ResourceMcpService) DoDelete(ctx context.Context, id string, _ *resources.McpServiceConfig) error {
	return r.client.AiGateway.DeleteMcpService(ctx, catalog.DeleteMcpServiceRequest{Name: mcpServiceNamePrefix + id})
}
