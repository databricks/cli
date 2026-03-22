package aitools

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var showSkills bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show installed AI skills version",
		RunE: func(cmd *cobra.Command, args []string) error {
			// showSkills is accepted for forward-compat but currently
			// skills is the only component, so the output is the same.
			_ = showSkills
			ctx := cmd.Context()

			globalDir, err := installer.GlobalSkillsDir(ctx)
			if err != nil {
				return err
			}

			state, err := installer.LoadState(globalDir)
			if err != nil {
				return fmt.Errorf("failed to load install state: %w", err)
			}

			if state == nil {
				cmdio.LogString(ctx, "No Databricks AI Tools components installed.")
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools install' to get started.")
				return nil
			}

			version := strings.TrimPrefix(state.Release, "v")
			skillNoun := "skills"
			if len(state.Skills) == 1 {
				skillNoun = "skill"
			}

			// Best-effort staleness check.
			if env.Get(ctx, "DATABRICKS_SKILLS_REF") != "" {
				cmdio.LogString(ctx, "Databricks AI Tools:")
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
				cmdio.LogString(ctx, "  Using custom ref: $DATABRICKS_SKILLS_REF")
				return nil
			}

			src := &installer.GitHubManifestSource{}
			latest, authoritative, err := src.FetchLatestRelease(ctx)
			if err != nil {
				log.Debugf(ctx, "Could not check for updates: %v", err)
				authoritative = false
			}

			cmdio.LogString(ctx, "Databricks AI Tools:")

			if !authoritative {
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
				cmdio.LogString(ctx, "  Could not check for latest version.")
				return nil
			}

			if latest == state.Release {
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s, up to date)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
			} else {
				latestVersion := strings.TrimPrefix(latest, "v")
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Update available: v"+latestVersion)
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools update' to update.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showSkills, "skills", false, "Show detailed skills version information")
	return cmd
}
