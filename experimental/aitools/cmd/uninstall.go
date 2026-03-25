package aitools

import (
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall all AI skills",
		Long: `Remove all installed Databricks AI skills from all coding agents.

Removes skill directories, symlinks, and the state file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installer.UninstallSkills(cmd.Context())
		},
	}
}
