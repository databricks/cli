package mcp

import (
	mcplib "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/experimental/apps-mcp/lib/server"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func NewMcpCmd() *cobra.Command {
	var warehouseID string
	var withWorkspaceTools bool

	cmd := &cobra.Command{
		Use:    "apps-mcp",
		Hidden: true,
		Short:  "Model Context Protocol server for AI agents",
		Long: `Start and manage an MCP server that provides AI agents with tools to interact with Databricks.

The MCP server exposes the following capabilities:
- Data exploration (query catalogs, schemas, tables, execute SQL)
- CLI command execution (bundle, apps, workspace operations)
- Workspace resource discovery

The server communicates via stdio using the Model Context Protocol.`,
		Example: `  # Start MCP server with required warehouse
  databricks experimental apps-mcp --warehouse-id abc123

  # Start with workspace tools enabled
  databricks experimental apps-mcp --warehouse-id abc123 --with-workspace-tools`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Build MCP config from flags
			cfg := &mcplib.Config{
				WithWorkspaceTools: withWorkspaceTools,
			}

			log.Infof(ctx, "Starting MCP server")

			// Create and start server with workspace client in context
			srv := server.NewServer(ctx, cfg)

			// Register tools
			if err := srv.RegisterTools(ctx); err != nil {
				log.Errorf(ctx, "Failed to register tools: %s", err)
				return err
			}

			// Run server
			return srv.Run(ctx)
		},
	}

	// Define flags
	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "Databricks SQL Warehouse ID")
	cmd.Flags().BoolVar(&withWorkspaceTools, "with-workspace-tools", false, "Enable workspace tools (file operations, bash, grep, glob)")

	cmd.AddCommand(newInstallCmd())

	return cmd
}
