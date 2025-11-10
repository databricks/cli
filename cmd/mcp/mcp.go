package mcp

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp",
		Short:  "[Experimental] Manage the Databricks CLI MCP server for coding agents",
		Hidden: true,
		Long: `Manage the Databricks CLI MCP (Model Context Protocol) server.

The MCP server enables coding agents like Claude Code and Cursor to interact
with Databricks, create projects, deploy bundles, run jobs, etc.

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║
╚════════════════════════════════════════════════════════════════╝

Common workflows:
  databricks mcp install   # Install in Claude Code or Cursor
  databricks mcp server    # Start server (used by agents)

Online documentation: https://docs.databricks.com/dev-tools/cli/mcp.html`,
		GroupID: "development",
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newToolCmd())

	return cmd
}
