package aitools

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aitools",
		Short:  "[Experimental] Manage AI tools server for coding agents",
		Hidden: true,
		Long: `Manage the Databricks CLI AI tools server (implements Model Context Protocol).

The AI tools server enables coding agents like Claude Code and Cursor to interact
with Databricks, create projects, deploy bundles, run jobs, etc.

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║
╚════════════════════════════════════════════════════════════════╝

Common workflows:
  databricks aitools install   # Install in Claude Code or Cursor
  databricks aitools server    # Start server (used by agents)

Online documentation: https://docs.databricks.com/dev-tools/cli/aitools.html`,
		GroupID: "development",
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newToolCmd())

	return cmd
}
