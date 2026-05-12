package aitools

import (
	"fmt"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var check, force, noNew bool
	var skillsFlag string
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update installed AI skills",
		Long: `Update installed Databricks AI skills to the latest release.

By default, updates all installed skills and auto-installs new skills
from the manifest. Use --no-new to skip new skills, or --check to
preview what would change without downloading.`,
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

			scopes, err := resolveScopeForUpdate(ctx, projectFlag, globalFlag, globalDir, projectDir)
			if err != nil {
				return err
			}

			installed := agents.DetectInstalled(ctx)
			src := &installer.GitHubManifestSource{}
			skills := splitAndTrim(skillsFlag)

			for _, scope := range scopes {
				if len(scopes) > 1 {
					cmdio.LogString(ctx, fmt.Sprintf("Updating %s skills...", scope))
				}

				opts := installer.UpdateOptions{
					Check: check,
					Force: force,
					NoNew: noNew,
					Scope: scope,
				}
				opts.Skills = skills

				result, err := installer.UpdateSkills(ctx, src, installed, opts)
				if err != nil {
					return err
				}
				if result != nil && (len(result.Updated) > 0 || len(result.Added) > 0) {
					cmdio.LogString(ctx, installer.FormatUpdateResult(result, check))
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Show what would be updated without downloading")
	cmd.Flags().BoolVar(&force, "force", false, "Re-download even if versions match")
	cmd.Flags().BoolVar(&noNew, "no-new", false, "Don't auto-install new skills from manifest")
	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to update (comma-separated)")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Update project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Update globally-scoped skills")
	return cmd
}
