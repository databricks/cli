package aitools

import (
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var skillsFlag string
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AI skills",
		Long: `Remove installed Databricks AI skills from all coding agents.

By default, removes all skills. Use --skills to remove specific skills only.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			globalDir, err := installer.GlobalSkillsDir(ctx)
			if err != nil {
				return err
			}
			projectDir, err := installer.ProjectSkillsDir(ctx)
			if err != nil {
				return err
			}

			scope, err := resolveScopeForUninstall(ctx, projectFlag, globalFlag, globalDir, projectDir)
			if err != nil {
				return err
			}

			opts := installer.UninstallOptions{
				Scope: scope,
			}
			opts.Skills = splitAndTrim(skillsFlag)
			return installer.UninstallSkillsOpts(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to uninstall (comma-separated)")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Uninstall project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Uninstall globally-scoped skills")
	return cmd
}
