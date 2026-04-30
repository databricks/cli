package aitools

import (
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aitools",
		Short: "Databricks AI Tools for coding agents",
		Long: `Manage Databricks AI Tools.

Provides commands to install, update, and manage Databricks skills for
detected coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub
Copilot, Antigravity).`,
	}

	cmd.AddCommand(NewInstallCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewUninstallCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewVersionCmd())

	return cmd
}
