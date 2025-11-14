package aitools

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aitools",
		Short:  "Manage AI agent skills and MCP server for coding agents",
		Hidden: true,
		Long: `Manage Databricks AI agent skills and MCP server.

This provides AI agents like Claude Code and Cursor with capabilities to interact
with Databricks, create projects, deploy bundles, run jobs, etc.

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║
╚════════════════════════════════════════════════════════════════╝

Common workflows:
  databricks experimental aitools install   # Install in Claude Code or Cursor
  databricks experimental aitools server    # Start server (used by agents)

Online documentation: https://docs.databricks.com/dev-tools/cli/aitools.html`,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newToolCmd())

	return cmd
}
