package aitools

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
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

			cmdio.LogString(ctx, "Databricks AI Tools:")

			latestRef := installer.GetSkillsRef(ctx)
			if latestRef == state.Release {
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s, up to date)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
			} else {
				latestVersion := strings.TrimPrefix(latestRef, "v")
				cmdio.LogString(ctx, fmt.Sprintf("  Skills: v%s (%d %s)", version, len(state.Skills), skillNoun))
				cmdio.LogString(ctx, "  Update available: v"+latestVersion)
				cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools update' to update.")
			}

			return nil
		},
	}
}
