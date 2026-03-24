package aitools

import (
	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var check, force, noNew bool
	var skillsFlag string

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
			installed := agents.DetectInstalled(ctx)
			src := &installer.GitHubManifestSource{}

			opts := installer.UpdateOptions{
				Check: check,
				Force: force,
				NoNew: noNew,
			}
			opts.Skills = splitAndTrim(skillsFlag)

			result, err := installer.UpdateSkills(ctx, src, installed, opts)
			if err != nil {
				return err
			}
			if result != nil && (len(result.Updated) > 0 || len(result.Added) > 0) {
				cmdio.LogString(ctx, installer.FormatUpdateResult(result, check))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Show what would be updated without downloading")
	cmd.Flags().BoolVar(&force, "force", false, "Re-download even if versions match")
	cmd.Flags().BoolVar(&noNew, "no-new", false, "Don't auto-install new skills from manifest")
	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to update (comma-separated)")
	return cmd
}
