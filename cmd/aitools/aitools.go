package aitools

import (
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aitools",
		Short: "Databricks AI Tools for coding agents",
		Long: `Install Databricks skills into your coding agent so it can work
effectively with Databricks resources (bundles, jobs, SQL, and more).

Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub
Copilot, Antigravity.`,
	}

	cmd.AddCommand(NewInstallCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewUninstallCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewVersionCmd())

	return cmd
}
