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
			Description: "Configure authentication for Databricks. Only call when Databricks authentication has has failed to authenticate automatically or when the user explicitly asks for using a specific host or profile. Validates credentials and stores the authenticated client in the session.",
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
			Description: "**REQUIRED DURING PLAN MODE** - Call this FIRST when planning ANY Databricks work. Use this to discover available workspaces, warehouses, and get workflow recommendations for your specific task. Even if you're just reading an assignment document, call this first. Especially important when task involves: creating Databricks projects/apps/pipelines/jobs, SQL pipelines or data transformation workflows, deploying code to multiple environments (dev/prod), or working with databricks.yml files. You DON'T need a workspace name - call this when starting ANY Databricks planning to understand workspace capabilities and recommended tooling before you create your plan.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args struct{}) (*mcpsdk.CallToolResult, any, error) {
			log.Debug(ctx, "explore called")
			result, err := Explore(session.WithSession(ctx, p.session))
			if err != nil {
				return nil, nil, err
			}
			return mcpsdk.CreateNewTextContentResult(result), nil, nil
		},
	)

	// Register invoke_databricks_cli tool
	type InvokeDatabricksCLIInput struct {
		Command string `json:"command" jsonschema:"required" jsonschema_description:"The full Databricks CLI command to run, e.g. 'bundle deploy' or 'bundle validate'. Do not include the 'databricks' prefix."`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name: "invoke_databricks_cli",
			Description: `Execute Databricks CLI command. Pass all arguments as a single string.

## ⚡ EFFICIENT DATA DISCOVERY (Recommended):
1. 'catalogs list' → find available catalogs
2. 'schemas list CATALOG' → find schemas in catalog
3. 'tables list CATALOG SCHEMA' → find tables in schema
4. 'experimental apps-mcp tools discover-schema TABLE1 TABLE2 TABLE3' → **BATCH discover multiple tables in ONE call** ⚡

## Data Commands:
- Execute SQL: 'experimental apps-mcp tools query "SELECT * FROM table LIMIT 5"' (returns JSON + row count)
- Discover schema: 'experimental apps-mcp tools discover-schema TABLE1 TABLE2 ...' (columns, types, samples, nulls)
  ↳ ALWAYS use batch mode: 'experimental apps-mcp tools discover-schema tbl1 tbl2 tbl3' instead of 3 separate calls
  ↳ Table format: CATALOG.SCHEMA.TABLE (e.g., samples.nyctaxi.trips)

## Project Commands:
- Init template: 'experimental apps-mcp tools init-template PROJECT_NAME' → create new app from tRPC template
- Bundle commands: 'bundle deploy', 'bundle validate', 'bundle run JOB_NAME'

## Common Errors:
❌ 'tables list samples.tpcds_sf1' → Wrong format!
✅ 'tables list samples tpcds_sf1' → Correct (CATALOG SCHEMA as separate args)

## Best Practices:
✅ Use batch discover-schema for multiple tables (faster)
✅ Test SQL with 'experimental apps-mcp tools query' before implementing in code
✅ Use 'experimental apps-mcp tools init-template' to scaffold new projects`,
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args InvokeDatabricksCLIInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "invoke_databricks_cli called: command=%s", args.Command)
			result, err := InvokeDatabricksCLI(ctx, args.Command)
			if err != nil {
				return nil, nil, err
			}
			return mcpsdk.CreateNewTextContentResult(result), nil, nil
		},
	)

	log.Infof(p.ctx, "Registered CLI tools: count=%d", 3)
	return nil
}
