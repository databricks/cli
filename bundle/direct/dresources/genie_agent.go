package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
)

// ResourceGenieAgent manages Genie Agents (the AI/BI Genie assistant surface).
// It mirrors ResourceGenieSpace, which manages the same backend object through
// the older Genie Spaces API; declaring the same object under both resource
// types is unsupported. Bind of an object created through either surface works,
// since both address it by the same id.
//
// API: https://docs.databricks.com/api/workspace/genieagents
// Terraform: databricks_genie_agent
//
// Identity: the engine tracks the bare genie_agent_id; the API addresses the
// resource by its full name (agents/{genie_agent_id}). genie_agent_id is
// optional on create (server-assigned when omitted) and immutable, so a local
// change recreates.
const genieAgentNamePrefix = "agents/"

type ResourceGenieAgent struct {
	client *databricks.WorkspaceClient
}

func (*ResourceGenieAgent) New(client *databricks.WorkspaceClient) *ResourceGenieAgent {
	return &ResourceGenieAgent{client: client}
}

func (*ResourceGenieAgent) PrepareState(input *resources.GenieAgent) *resources.GenieAgentConfig {
	return &input.GenieAgentConfig
}

// genieAgentBody builds the GenieAgent write payload from the bundle config.
// `name` is server-derived and the create/update paths carry the id, so it is
// never sent; the remaining fields are output-only and stay zero.
func genieAgentBody(config *resources.GenieAgentConfig) genieagents.GenieAgent {
	body := genieagents.GenieAgent{
		DisplayName: config.DisplayName,
		Description: config.Description,
		WarehouseId: config.WarehouseId,
		ParentPath:  config.ParentPath,

		// A nil config definition marshals as the zero value, which the API
		// rejects (the definition must set at least one field).
		Definition: genieagents.GenieAgentDefinition{
			Benchmarks:   nil,
			Config:       nil,
			DataSources:  nil,
			Instructions: nil,
		},

		// Output-only fields; `name` is carried by the request path.
		Name:                "",
		Etag:                "",
		CreateTime:          nil,
		UpdateTime:          nil,
		EffectiveParentPath: "",
		ForceSendFields:     nil,
	}
	if config.Definition != nil {
		body.Definition = *config.Definition
	}
	return body
}

// responseToGenieAgentConfig converts a returned GenieAgent into the bundle
// state. The server-derived name, etag, timestamps and effective_parent_path
// have no config counterpart and are dropped here.
func responseToGenieAgentConfig(agent *genieagents.GenieAgent) *resources.GenieAgentConfig {
	definition := agent.Definition
	return &resources.GenieAgentConfig{
		GenieAgentId:    strings.TrimPrefix(agent.Name, genieAgentNamePrefix),
		DisplayName:     agent.DisplayName,
		Description:     agent.Description,
		Definition:      &definition,
		WarehouseId:     agent.WarehouseId,
		ParentPath:      agent.ParentPath,
		ForceSendFields: nil,
	}
}

func (r *ResourceGenieAgent) DoRead(ctx context.Context, id string) (*resources.GenieAgentConfig, error) {
	agent, err := r.client.GenieAgents.GetGenieAgent(ctx, genieagents.GetGenieAgentRequest{
		Name: genieAgentNamePrefix + id,
	})
	if err != nil {
		return nil, err
	}
	return responseToGenieAgentConfig(agent), nil
}

// DoCreate deliberately issues up to three SDK calls — create, mkdirs, create —
// an exception to the one-SDK-call-per-DoCreate rule in
// bundle/direct/dresources/README.md. The default parent_path is the bundle's
// workspace resource folder (workspace.resource_path), which does not exist on
// a first deploy, so create retries once after mkdirs. ResourceGenieSpace.DoCreate
// handles the same condition the same way; isMissingGenieParentPathError
// recognises the two shapes the Genie backend uses for a missing folder. That
// the Agents API surfaces one of those shapes is confirmed by the lifecycle
// test's cloud run.
func (r *ResourceGenieAgent) DoCreate(ctx context.Context, config *resources.GenieAgentConfig) (string, *resources.GenieAgentConfig, error) {
	req := genieagents.CreateGenieAgentRequest{
		GenieAgent:      genieAgentBody(config),
		GenieAgentId:    config.GenieAgentId,
		ForceSendFields: nil,
	}

	resp, err := r.client.GenieAgents.CreateGenieAgent(ctx, req)
	if err != nil && isMissingGenieParentPathError(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		resp, err = r.client.GenieAgents.CreateGenieAgent(ctx, req)
	}
	if err != nil {
		return "", nil, err
	}

	return strings.TrimPrefix(resp.Name, genieAgentNamePrefix), responseToGenieAgentConfig(resp), nil
}

// DoUpdate sends update_mask "*": the API documents that the wildcard replaces
// every updatable field (display_name, description, warehouse_id, definition,
// parent_path), taking each field's value from the body even when unset there,
// so removing a field from config clears it. definition is replaced as a whole;
// paths within it are not supported. The lifecycle acceptance test exercises
// this mask against a real workspace.
//
// Etag is intentionally left empty (here and in DoDelete): it guards concurrent
// definition edits, which a serial bundle deploy does not have. Drift is
// detected on read by comparing the full config against the GET response,
// matching ResourceMcpService.
func (r *ResourceGenieAgent) DoUpdate(ctx context.Context, id string, config *resources.GenieAgentConfig, _ *PlanEntry) (*resources.GenieAgentConfig, error) {
	resp, err := r.client.GenieAgents.UpdateGenieAgent(ctx, genieagents.UpdateGenieAgentRequest{
		GenieAgent:      genieAgentBody(config),
		Name:            genieAgentNamePrefix + id,
		UpdateMask:      fieldmask.FieldMask{Paths: []string{"*"}},
		Etag:            "",
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}
	return responseToGenieAgentConfig(resp), nil
}

func (r *ResourceGenieAgent) DoDelete(ctx context.Context, id string, _ *resources.GenieAgentConfig) error {
	// Delete moves the agent to trash; NOT_FOUND afterwards means it is gone
	// (the trash is not readable through this API).
	return r.client.GenieAgents.DeleteGenieAgent(ctx, genieagents.DeleteGenieAgentRequest{
		Name:            genieAgentNamePrefix + id,
		Etag:            "",
		ForceSendFields: nil,
	})
}
