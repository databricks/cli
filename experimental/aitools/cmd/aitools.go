package aitools

import (
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aitools",
		Hidden: true,
		Short:  "Databricks AI Tools for coding agents",
		Long: `Manage Databricks AI Tools.

Provides commands to:
- Install the AI tools in coding agents (install)
- Manage skills (skills)
- Access tools directly (tools)`,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newSkillsCmd())
	cmd.AddCommand(newToolsCmd())

	return cmd
}
