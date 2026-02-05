package sdkdocs

import (
	"context"

	mcp "github.com/databricks/cli/experimental/aitools/lib"
	mcpsdk "github.com/databricks/cli/experimental/aitools/lib/mcp"
	"github.com/databricks/cli/experimental/aitools/lib/providers"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/log"
)

func init() {
	providers.Register("sdkdocs", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, cfg, sess)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider provides SDK documentation search capabilities.
type Provider struct {
	config  *mcp.Config
	session *session.Session
	ctx     context.Context
	index   *SDKDocsIndex
}

// NewProvider creates a new SDK docs provider.
func NewProvider(ctx context.Context, cfg *mcp.Config, sess *session.Session) (*Provider, error) {
	index, err := LoadIndex()
	if err != nil {
		log.Warnf(ctx, "Failed to load SDK docs index: %v", err)
		// Return a provider with an empty index rather than failing
		index = &SDKDocsIndex{
			Services: make(map[string]*ServiceDoc),
			Types:    make(map[string]*TypeDoc),
			Enums:    make(map[string]*EnumDoc),
		}
	}

	log.Infof(ctx, "SDK docs provider initialized: %d services, %d types, %d enums",
		len(index.Services), len(index.Types), len(index.Enums))

	return &Provider{
		config:  cfg,
		session: sess,
		ctx:     ctx,
		index:   index,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "sdkdocs"
}

// RegisterTools registers the SDK documentation tools with the MCP server.
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering SDK docs tools")

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name: "databricks_query_sdk_docs",
			Description: `Search Databricks SDK documentation for methods, types, and examples.

Use this tool to find:
- API methods: "how to create a job", "list clusters", "run pipeline"
- Type definitions: "JobSettings fields", "ClusterSpec parameters"
- Enums: "run lifecycle states", "cluster state values"

Returns method signatures, parameter descriptions, return types, and usage examples.
This is useful when you need to understand the correct way to call Databricks APIs.`,
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args QuerySDKDocsInput) (*mcpsdk.CallToolResult, any, error) {
			return p.querySDKDocs(ctx, args)
		},
	)

	log.Infof(p.ctx, "Registered SDK docs tools: count=%d", 1)
	return nil
}
