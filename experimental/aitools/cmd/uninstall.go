package aitools

import (
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var skillsFlag string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AI skills",
		Long: `Remove installed Databricks AI skills from all coding agents.

By default, removes all skills. Use --skills to remove specific skills only.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := installer.UninstallOptions{}
			opts.Skills = splitAndTrim(skillsFlag)
			return installer.UninstallSkillsOpts(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to uninstall (comma-separated)")
	return cmd
}
