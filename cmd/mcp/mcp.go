package mcp

import (
	"fmt"

	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage the Databricks CLI MCP server for coding agents",
		Long: `Manage the Databricks CLI MCP (Model Context Protocol) server for coding agents.

The MCP server enables coding agents like Claude Code and Cursor to interact with
Databricks directly, allowing them to create projects, deploy bundles, and run jobs.

Common workflows:
  databricks mcp           # Install MCP server in coding agents
  databricks mcp server    # Start the MCP server (used by coding agents)
  databricks mcp uninstall # Uninstall instructions

Online documentation: https://docs.databricks.com/dev-tools/cli/mcp.html`,
		GroupID: "development",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Check subcommand
			if len(args) > 0 {
				switch args[0] {
				case "server":
					return runServer(ctx)
				case "uninstall":
					return runUninstall(ctx)
				default:
					return fmt.Errorf("unknown subcommand: %s", args[0])
				}
			}

			// Default: install
			return runInstall(ctx)
		},
	}

	return cmd
}
