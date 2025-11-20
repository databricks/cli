package mcp

import (
	"errors"
	"os"

	"github.com/databricks/cli/cmd/root"
	mcplib "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/experimental/apps-mcp/lib/server"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func NewMcpCmd() *cobra.Command {
	var warehouseID string
	var allowDeployment bool
	var withWorkspaceTools bool

	cmd := &cobra.Command{
		Use:    "apps-mcp",
		Hidden: true,
		Short:  "Model Context Protocol server for AI agents",
		Long: `Start and manage an MCP server that provides AI agents with tools to interact with Databricks.

The MCP server exposes the following capabilities:
- Databricks integration (query catalogs, schemas, tables, execute SQL)
- Project scaffolding (generate full-stack TypeScript applications)
- Sandboxed execution (isolated file/command execution)

The server communicates via stdio using the Model Context Protocol.`,
		Example: `  # Start MCP server with required warehouse
  databricks experimental apps-mcp --warehouse-id abc123

  # Start with workspace tools enabled
  databricks experimental apps-mcp --warehouse-id abc123 --with-workspace-tools

  # Start with deployment tools enabled
  databricks experimental apps-mcp --warehouse-id abc123 --allow-deployment`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if warehouseID == "" {
				warehouseID = os.Getenv("DATABRICKS_WAREHOUSE_ID")
				if warehouseID == "" {
					return errors.New("DATABRICKS_WAREHOUSE_ID environment variable is required")
				}
			}

			w := cmdctx.WorkspaceClient(ctx)

			// Build MCP config from flags
			cfg := &mcplib.Config{
				AllowDeployment:    allowDeployment,
				WithWorkspaceTools: withWorkspaceTools,
				WarehouseID:        warehouseID,
				DatabricksHost:     w.Config.Host,
				IoConfig: &mcplib.IoConfig{
					Validation: &mcplib.ValidationConfig{},
				},
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
	cmd.Flags().BoolVar(&allowDeployment, "allow-deployment", false, "Enable deployment tools")
	cmd.Flags().BoolVar(&withWorkspaceTools, "with-workspace-tools", false, "Enable workspace tools (file operations, bash, grep, glob)")

	return cmd
}
