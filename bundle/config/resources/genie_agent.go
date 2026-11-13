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
// It mirrors ResourceMcpService: the SDK models the create input `genie_agent_id`
// as a URL parameter (`json:"-"`) outside the GenieAgent body and derives the
// resource `name` (`agents/{genie_agent_id}`) server-side, so we expose a flat
// struct with the identity plus the mutable body. The server-derived `name`,
// `etag`, timestamps and `effective_parent_path` stay out of the config type;
// the engine tracks the bare `genie_agent_id` as the resource id.
type GenieAgentConfig struct {
	// Client-specified ID for the agent: 32 lowercase hexadecimal characters
	// (a UUID without dashes), not all zeros. Assigned by the server if
	// omitted. Immutable: changing it recreates the resource.
	GenieAgentId string `json:"genie_agent_id,omitempty"`
	// Display name of the Genie Agent. Must be unique within its folder. The
	// maximum length is 63 characters.
	DisplayName string `json:"display_name"`
	// Description of the Genie Agent. The maximum length is 4,096 characters.
	Description string `json:"description,omitempty"`
	// Warehouse associated with the Genie Agent.
	WarehouseId string `json:"warehouse_id"`
	// The workspace folder to create the Genie Agent in. If omitted, the
	// caller's home folder is used. On update, setting it moves the agent and
	// clearing it does not.
	ParentPath string `json:"parent_path,omitempty"`
	// The definition of the Genie Agent: its sample questions, data sources,
	// instructions, and benchmarks. Must set at least one field. Replaced as a
	// whole on update.
	Definition *genieagents.GenieAgentDefinition `json:"definition"`

	ForceSendFields []string `json:"-" url:"-"`
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
