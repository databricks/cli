package mcp

import (
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	mcplib "github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/server"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var warehouseID string
	var allowDeployment bool
	var dockerImage string
	var useDagger bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the MCP server",
		Long:  "Start the Model Context Protocol server to provide Databricks tools to AI agents.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Get Databricks client from context
			w := cmdctx.WorkspaceClient(ctx)

			// Build MCP config from flags
			cfg := &mcplib.Config{
				AllowDeployment:    allowDeployment,
				WithWorkspaceTools: true,
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

			// Create and start server
			srv := server.NewServer(cfg, ctx)

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
	cmd.Flags().StringVar(&dockerImage, "docker-image", "node:20-alpine", "Docker image for validation")
	cmd.Flags().BoolVar(&useDagger, "use-dagger", true, "Use Dagger for containerized validation")

	return cmd
}
