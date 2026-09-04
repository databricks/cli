package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// McpServiceConfig is the bundle-authored state for an AI Gateway MCP service.
//
// It mirrors ModelServiceConfig: the SDK models the create inputs `parent` and
// `mcp_service_id` as URL parameters (`json:"-"`) outside the McpService body
// and derives the resource `name`
// (`mcp-services/{catalog}.{schema}.{mcp_service}`) server-side, so we expose a
// flat struct with the immutable identity plus the mutable body. Owner is not
// exposed yet (the API returns effective_owner on read, not owner).
type McpServiceConfig struct {
	// Parent schema, format `schemas/{catalog}.{schema}`. Immutable: the server
	// derives `name` from parent + mcp_service_id, so changing it recreates the
	// resource.
	Parent string `json:"parent"`
	// Leaf id of the MCP service, e.g. "my_mcp_service". Immutable.
	McpServiceId string `json:"mcp_service_id"`
	// User-provided description.
	Comment string `json:"comment,omitempty"`
	// Operational configuration: source connection, tool selectors, rate limit.
	Config *catalog.McpServiceConfig `json:"config,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func (c *McpServiceConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c McpServiceConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type McpService struct {
	BaseResource
	McpServiceConfig
}

// UnmarshalJSON / MarshalJSON are defined on the wrapper so it does not inherit
// McpServiceConfig's promoted marshaler, which would silently drop the
// BaseResource fields (id, url, lifecycle, modified_status).
func (m *McpService) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, m)
}

func (m McpService) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(m)
}

func (m *McpService) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	// The engine tracks the id as the bare {catalog}.{schema}.{mcp_service};
	// the API addresses the resource by its full name.
	_, err := w.AiGateway.GetMcpService(ctx, catalog.GetMcpServiceRequest{Name: "mcp-services/" + id})
	if err != nil {
		log.Debugf(ctx, "mcp service %s does not exist", id)
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*McpService) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "mcp_service",
		PluralName:    "mcp_services",
		SingularTitle: "MCP service",
		PluralTitle:   "MCP services",
	}
}

func (m *McpService) InitializeURL(baseURL url.URL) {
	if m.ID == "" {
		return
	}
	// The id is the bare {catalog}.{schema}.{mcp_service}; ResourceURL splits it
	// into the Catalog Explorer path explore/data/mcp-services/...
	m.URL = workspaceurls.ResourceURL(baseURL, "mcp_services", m.ID)
}

func (m *McpService) GetName() string {
	return m.McpServiceId
}
