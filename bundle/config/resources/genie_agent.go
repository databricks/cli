package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
)

// GenieAgentConfig is the bundle-authored state for a Genie Agent.
//
// It embeds the SDK body. The SDK models the create input `genie_agent_id` as a
// URL parameter (`json:"-"`) outside the body and derives the resource `name`
// (`agents/{genie_agent_id}`) server-side, so the ID is added here. The
// output-only SDK fields (etag, create_time, update_time, effective_parent_path)
// are removed from the bundle schema by the generator and zeroed by PrepareState.
type GenieAgentConfig struct {
	genieagents.GenieAgent

	// Client-specified ID for the agent: 32 lowercase hexadecimal characters
	// (a UUID without dashes), not all zeros. Assigned by the server if
	// omitted. Immutable: changing it recreates the resource. After bundle destroy
	// or deletion, the agent remains in trash with its ID reserved until
	// permanently removed. Omit this field to let the server assign an ID.
	GenieAgentId string `json:"genie_agent_id,omitempty"`
}

func (c *GenieAgentConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c GenieAgentConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

// GenieAgent is a Genie Agent resource (the AI/BI Genie assistant surface).
// It manages the same backend object as the older genie_spaces resource, so
// declaring the same object under both resource types is unsupported.
type GenieAgent struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	GenieAgentConfig

	Permissions []Permission `json:"permissions,omitempty"`
}

func (r *GenieAgent) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r GenieAgent) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

func (*GenieAgent) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	// The engine tracks the id as the bare genie_agent_id; the API addresses
	// the resource by its full name.
	_, err := w.GenieAgents.GetGenieAgent(ctx, genieagents.GetGenieAgentRequest{
		Name: "agents/" + id,
	})
	if err != nil {
		if apierr.IsMissing(err) {
			log.Debugf(ctx, "genie agent %s does not exist", id)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*GenieAgent) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "genie_agent",
		PluralName:    "genie_agents",
		SingularTitle: "Genie Agent",
		PluralTitle:   "Genie Agents",
	}
}

func (r *GenieAgent) InitializeURL(baseURL url.URL) {
	if r.ID == "" {
		return
	}

	// Agents and spaces are the same backend object, so an agent's UI page is
	// the same Genie room page a space uses.
	r.URL = workspaceurls.ResourceURL(baseURL, "genie_agents", r.ID)
}

func (r *GenieAgent) GetName() string {
	return r.DisplayName
}
