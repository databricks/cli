package mcp

import (
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
	var dockerImage string
	var useDagger bool

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
  databricks experimental apps-mcp --warehouse-id abc123 --allow-deployment

  # Start with custom Docker image for validation
  databricks experimental apps-mcp --warehouse-id abc123 --docker-image node:20-alpine

  # Start without containerized validation
  databricks experimental apps-mcp --warehouse-id abc123 --use-dagger=false`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			w := cmdctx.WorkspaceClient(ctx)

			// Build MCP config from flags
			cfg := &mcplib.Config{
				AllowDeployment:    allowDeployment,
				WithWorkspaceTools: withWorkspaceTools,
				WarehouseID:        warehouseID,
				DatabricksHost:     w.Config.Host,
				IoConfig: &mcplib.IoConfig{
					Validation: &mcplib.ValidationConfig{
						DockerImage: dockerImage,
						UseDagger:   useDagger,
					},
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
	cmd.Flags().StringVar(&dockerImage, "docker-image", "node:20-alpine", "Docker image for validation")
	cmd.Flags().BoolVar(&useDagger, "use-dagger", true, "Use Dagger for containerized validation")

	cmd.AddCommand(newCheckCmd())

	return cmd
}
