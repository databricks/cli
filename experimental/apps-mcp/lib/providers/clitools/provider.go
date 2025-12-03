package clitools

import (
	"context"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/log"
)

func init() {
	providers.Register("clitools", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, cfg, sess)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider represents the CLI provider that registers MCP tools for CLI operations
type Provider struct {
	config  *mcp.Config
	session *session.Session
	ctx     context.Context
}

// NewProvider creates a new CLI provider
func NewProvider(ctx context.Context, cfg *mcp.Config, sess *session.Session) (*Provider, error) {
	return &Provider{
		config:  cfg,
		session: sess,
		ctx:     ctx,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "clitools"
}

// RegisterTools registers all CLI tools with the MCP server
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering CLI tools")

	// Register databricks_configure_auth
	type ConfigureAuthInput struct {
		Host    *string `json:"host,omitempty" jsonschema_description:"Databricks workspace URL (e.g., https://example.cloud.databricks.com). If not provided, uses default from environment or config file"`
		Profile *string `json:"profile,omitempty" jsonschema_description:"Profile name from ~/.databrickscfg. If not provided, uses default profile"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_configure_auth",
			Description: "Configure authentication for Databricks. Only call when Databricks authentication has failed to authenticate automatically or when the user explicitly asks for using a specific host or profile. Validates credentials and stores the authenticated client in the session.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args ConfigureAuthInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debug(ctx, "databricks_configure_auth called")

			sess, err := session.GetSession(ctx)
			if err != nil {
				return nil, nil, err
			}

			client, err := ConfigureAuth(ctx, sess, args.Host, args.Profile)
			if err != nil {
				return nil, nil, err
			}

			message := "Authentication configured successfully"
			if args.Host != nil {
				message += " for host: " + *args.Host
			}
			if args.Profile != nil {
				message += " using profile: " + *args.Profile
			}

			// Get user info to confirm auth
			me, err := client.CurrentUser.Me(ctx)
			if err == nil && me.UserName != "" {
				message += "\nAuthenticated as: " + me.UserName
			}

			return mcpsdk.CreateNewTextContentResult(message), nil, nil
		},
	)

	// Register explore tool
	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "explore",
			Description: "Discover available Databricks workspaces, warehouses, and get workflow recommendations. Call this FIRST when planning ANY Databricks work involving apps, pipelines, jobs, bundles, or SQL workflows. Returns workspace capabilities and recommended tooling.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args struct{}) (*mcpsdk.CallToolResult, any, error) {
			log.Debug(ctx, "explore called")
			result, err := Explore(ctx)
			if err != nil {
				return nil, nil, err
			}
			return mcpsdk.CreateNewTextContentResult(result), nil, nil
		},
	)

	// Register invoke_databricks_cli tool
	type InvokeDatabricksCLIInput struct {
		WorkingDirectory string   `json:"working_directory" jsonschema:"required" jsonschema_description:"The directory to run the command in."`
		Args             []string `json:"args" jsonschema:"required" jsonschema_description:"CLI arguments as array, e.g. [\"bundle\", \"deploy\"] or [\"bundle\", \"validate\", \"--target\", \"dev\"]. Do not include 'databricks' prefix."`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "invoke_databricks_cli",
			Description: "Execute Databricks CLI command. Pass arguments as an array of strings.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args InvokeDatabricksCLIInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "invoke_databricks_cli called: args=%v, working_directory=%s", args.Args, args.WorkingDirectory)
			result, err := InvokeDatabricksCLI(ctx, args.Args, args.WorkingDirectory)
			if err != nil {
				return nil, nil, err
			}
			return mcpsdk.CreateNewTextContentResult(result), nil, nil
		},
	)

	log.Infof(p.ctx, "Registered CLI tools: count=%d", 3)
	return nil
}
