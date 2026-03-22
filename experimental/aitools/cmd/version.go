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
	return &cobra.Command{
		Use:   "version",
		Short: "Show installed AI skills version",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				cmdio.LogString(ctx, "Databricks AI skills: not installed")
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools install' to install.")
				return nil
			}

			version := strings.TrimPrefix(state.Release, "v")
			cmdio.LogString(ctx, fmt.Sprintf("Databricks AI skills v%s", version))
			cmdio.LogString(ctx, fmt.Sprintf("  Skills installed: %d", len(state.Skills)))
			cmdio.LogString(ctx, fmt.Sprintf("  Last updated:     %s", state.LastUpdated.Format("2006-01-02")))

			// Best-effort staleness check.
			if env.Get(ctx, "DATABRICKS_SKILLS_REF") != "" {
				cmdio.LogString(ctx, "  Using custom ref:  $DATABRICKS_SKILLS_REF")
				return nil
			}

			src := &installer.GitHubManifestSource{}
			latest, err := src.FetchLatestRelease(ctx)
			if err != nil {
				log.Debugf(ctx, "Could not check for updates: %v", err)
				return nil
			}

			if latest == state.Release {
				cmdio.LogString(ctx, "  Status:           up to date")
			} else {
				latestVersion := strings.TrimPrefix(latest, "v")
				cmdio.LogString(ctx, fmt.Sprintf("  Status:           update available (v%s)", latestVersion))
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools update' to update.")
			}

			return nil
		},
	}
}
