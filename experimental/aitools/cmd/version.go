package aitools

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show installed AI skills version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			globalDir, err := installer.GlobalSkillsDir(ctx)
			if err != nil {
				return err
			}
			globalState, err := installer.LoadState(globalDir)
			if err != nil {
				return fmt.Errorf("failed to load global install state: %w", err)
			}

			// Try loading project state (may fail if not in a project, that's ok).
			var projectState *installer.InstallState
			projectDir, projErr := installer.ProjectSkillsDir(ctx)
			if projErr == nil {
				projectState, err = installer.LoadState(projectDir)
				if err != nil {
					return fmt.Errorf("failed to load project install state: %w", err)
				}
			}

			if globalState == nil && projectState == nil {
				cmdio.LogString(ctx, "No Databricks AI Tools components installed.")
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks experimental aitools install' to get started.")
				return nil
			}

			latestRef := installer.GetSkillsRef(ctx)
			bothScopes := globalState != nil && projectState != nil

			cmdio.LogString(ctx, "Databricks AI Tools:")

			if globalState != nil {
				label := "Skills"
				if bothScopes {
					label = "Skills (global)"
				}
				printVersionLine(ctx, label, globalState, latestRef)
			}

			if projectState != nil {
				label := "Skills"
				if bothScopes {
					label = "Skills (project)"
				}
				printVersionLine(ctx, label, projectState, latestRef)
			}

			return nil
		},
	}

	return cmd
}

// printVersionLine prints a single version line for a scope.
func printVersionLine(ctx context.Context, label string, state *installer.InstallState, latestRef string) {
	version := strings.TrimPrefix(state.Release, "v")
	skillNoun := "skills"
	if len(state.Skills) == 1 {
		skillNoun = "skill"
	}

	if latestRef == state.Release {
		cmdio.LogString(ctx, fmt.Sprintf("  %s: v%s (%d %s, up to date)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
	} else {
		latestVersion := strings.TrimPrefix(latestRef, "v")
		cmdio.LogString(ctx, fmt.Sprintf("  %s: v%s (%d %s)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Update available: v"+latestVersion)
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Run 'databricks experimental aitools update' to update.")
	}
}
