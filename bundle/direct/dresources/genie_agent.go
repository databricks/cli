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
	state := input.GenieAgentConfig
	// Server-derived fields are never user input. Zeroing them keeps the saved
	// state and local diffs to the authored body; their remote values are read by
	// DoRead and skipped via ignore_remote_changes.
	state.Name = ""
	state.Etag = ""
	state.CreateTime = nil
	state.UpdateTime = nil
	state.EffectiveParentPath = ""
	return &state
}

// responseToGenieAgentConfig converts a returned GenieAgent into remote state.
// The API returns parent_path exactly as submitted; effective_parent_path may
// omit /Workspace.
func responseToGenieAgentConfig(agent *genieagents.GenieAgent) *resources.GenieAgentConfig {
	return &resources.GenieAgentConfig{
		GenieAgent:   *agent,
		GenieAgentId: strings.TrimPrefix(agent.Name, genieAgentNamePrefix),
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

// DoCreate retries once after creating the default parent folder
// (${workspace.resource_path}), which does not exist on a first deploy.
// ResourceGenieSpace.DoCreate handles the same condition the same way.
func (r *ResourceGenieAgent) DoCreate(ctx context.Context, config *resources.GenieAgentConfig) (string, *resources.GenieAgentConfig, error) {
	req := genieagents.CreateGenieAgentRequest{
		GenieAgent:      config.GenieAgent,
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
// paths within it are not supported.
//
// The etag is the one from the remote read the plan was built on, so the update
// fails if the definition was edited after planning instead of overwriting it.
// Metadata edits do not change the etag, so they never trigger this. Drift is
// otherwise detected on read by comparing the full config against the GET response.
func (r *ResourceGenieAgent) DoUpdate(ctx context.Context, id string, config *resources.GenieAgentConfig, planEntry *PlanEntry) (*resources.GenieAgentConfig, error) {
	resp, err := r.client.GenieAgents.UpdateGenieAgent(ctx, genieagents.UpdateGenieAgentRequest{
		GenieAgent:      config.GenieAgent,
		Name:            genieAgentNamePrefix + id,
		UpdateMask:      fieldmask.FieldMask{Paths: []string{"*"}},
		Etag:            plannedGenieAgentEtag(planEntry),
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}
	return responseToGenieAgentConfig(resp), nil
}

// plannedGenieAgentEtag returns the etag from the plan's remote read. An empty
// etag sends no precondition, which only happens when the plan carries no remote state.
func plannedGenieAgentEtag(planEntry *PlanEntry) string {
	if planEntry == nil {
		return ""
	}
	remote, ok := planEntry.RemoteState.(*resources.GenieAgentConfig)
	if !ok || remote == nil {
		return ""
	}
	return remote.Etag
}

// DoDelete sends no etag. The engine passes only the persisted state here, and
// PrepareState does not persist the etag, so no plan-time etag is available. Using
// the etag from the last deploy instead would make destroy fail after any
// definition edit made since then.
func (r *ResourceGenieAgent) DoDelete(ctx context.Context, id string, _ *resources.GenieAgentConfig) error {
	// Delete moves the agent to trash; NOT_FOUND afterwards means it is gone
	// (the trash is not readable through this API).
	return r.client.GenieAgents.DeleteGenieAgent(ctx, genieagents.DeleteGenieAgentRequest{
		Name:            genieAgentNamePrefix + id,
		Etag:            "",
		ForceSendFields: nil,
	})
}
