package aitools

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
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
				cmdio.LogString(ctx, "Run 'databricks aitools install' to get started.")
				return nil
			}

			latestRef, _, err := installer.GetSkillsRef(ctx)
			if err != nil {
				log.Debugf(ctx, "could not resolve skills version: %v", err)
			}
			bothScopes := globalState != nil && projectState != nil

			cmdio.LogString(ctx, "Databricks AI Tools:")

			if globalState != nil {
				label := "Skills"
				if bothScopes {
					label = "Skills (global)"
				}
				if len(globalState.Skills) > 0 {
					printVersionLine(ctx, label, globalState, latestRef)
				}
				printPluginLines(ctx, globalState)
			}

			if projectState != nil {
				label := "Skills"
				if bothScopes {
					label = "Skills (project)"
				}
				if len(projectState.Skills) > 0 {
					printVersionLine(ctx, label, projectState, latestRef)
				}
				printPluginLines(ctx, projectState)
			}

			return nil
		},
	}

	return cmd
}

// printPluginLines prints one line per plugin recorded in the scope's state.
func printPluginLines(ctx context.Context, state *installer.InstallState) {
	for _, name := range slices.Sorted(maps.Keys(state.Plugins)) {
		rec := state.Plugins[name]
		cmdio.LogString(ctx, fmt.Sprintf("  Plugin (%s): v%s", agentDisplayName(name), rec.Version))
	}
}

// agentDisplayName returns the agent's display name, falling back to its
// registry name when it isn't a known agent.
func agentDisplayName(name string) string {
	if agent := agents.ByName(name); agent != nil {
		return agent.DisplayName
	}
	return name
}

// printVersionLine prints a single version line for a scope.
func printVersionLine(ctx context.Context, label string, state *installer.InstallState, latestRef string) {
	version := strings.TrimPrefix(state.Release, "v")
	skillNoun := "skills"
	if len(state.Skills) == 1 {
		skillNoun = "skill"
	}

	if latestRef == "" {
		cmdio.LogString(ctx, fmt.Sprintf("  %s: v%s (%d %s)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
		return
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
		cmdio.LogString(ctx, "Run 'databricks aitools update' to update.")
	}
}
