package mcp

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	mcplib "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/experimental/apps-mcp/lib/server"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func NewMcpCmd() *cobra.Command {
	var warehouseID string

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
  databricks experimental apps-mcp --warehouse-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create cancellable context for graceful shutdown
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// Handle shutdown signals (SIGINT, SIGTERM)
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// Build MCP config from flags
			cfg := &mcplib.Config{}

			log.Infof(ctx, "Starting MCP server")

			// Create and start server with workspace client in context
			srv := server.NewServer(ctx, cfg)

			// Register tools
			if err := srv.RegisterTools(ctx); err != nil {
				log.Errorf(ctx, "Failed to register tools: %s", err)
				return err
			}

			// Run server in goroutine so we can handle signals
			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.Run(ctx)
			}()

			// Wait for either server error or shutdown signal
			select {
			case err := <-errCh:
				// Server stopped (EOF, error, or context cancelled)
				if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
					log.Warnf(ctx, "Shutdown error: %v", shutdownErr)
				}
				return err
			case sig := <-sigCh:
				// Received shutdown signal - exit gracefully
				log.Infof(ctx, "Received signal %v, shutting down gracefully", sig)
				cancel() // Cancel context to stop server.Run()
				if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
					log.Warnf(ctx, "Shutdown error: %v", shutdownErr)
				}
				return nil
			}
		},
	}

	// Define flags
	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "Databricks SQL Warehouse ID")

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newToolsCmd())

	return cmd
}
