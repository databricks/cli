package mcp

import (
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Alias for skills install",
		Long: `Alias for "databricks experimental aitools skills install".

Installs Databricks skills for detected coding agents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsInstall(cmd.Context(), args)
		},
	}
}
