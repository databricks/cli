package aitools

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show installed skills and plugins version",
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
				cmdio.LogString(ctx, "No Databricks skills or plugins installed.")
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "Run 'databricks aitools install' to get started.")
				return nil
			}

			latestRef, _, err := installer.GetSkillsRef(ctx)
			if err != nil {
				log.Debugf(ctx, "could not resolve skills version: %v", err)
			}
			cmdio.LogString(ctx, "Databricks skills and plugins:")

			// Always label the scope (global / project) for both skills and
			// plugins so it is unambiguous where each thing is installed.
			if globalState != nil {
				if len(globalState.Skills) > 0 {
					printVersionLine(ctx, "Skills (global)", globalState, latestRef)
				}
				printPluginLines(ctx, globalState, "global")
			}

			if projectState != nil {
				if len(projectState.Skills) > 0 {
					printVersionLine(ctx, "Skills (project)", projectState, latestRef)
				}
				printPluginLines(ctx, projectState, "project")
			}

			return nil
		},
	}

	return cmd
}

// printPluginLines prints one line per plugin recorded in the scope's state,
// labeled with the scope so it is clear where the plugin is installed.
func printPluginLines(ctx context.Context, state *installer.InstallState, scope string) {
	for _, name := range slices.Sorted(maps.Keys(state.Plugins)) {
		rec := state.Plugins[name]
		scopeLabel := scope
		if rec.Scope != "" {
			scopeLabel += ", " + rec.Scope + " scope"
		}
		cmdio.LogString(ctx, fmt.Sprintf("  Plugin (%s, %s): %s", agentDisplayName(name), scopeLabel, versionToken(rec.Version)))
	}
}

// versionToken renders a skills/plugin version for output: the "latest"
// sentinel is explicit about tracking main; a concrete version gets a leading "v".
func versionToken(v string) string {
	if v == "" {
		return "version unknown"
	}
	if v == "latest" {
		return "latest (tracking main)"
	}
	if !semver.IsValid("v" + v) {
		return v
	}
	return "v" + v
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
	version := versionToken(installer.DisplaySkillsVersion(state.Release))
	skillNoun := "skills"
	if len(state.Skills) == 1 {
		skillNoun = "skill"
	}

	if latestRef == "" {
		cmdio.LogString(ctx, fmt.Sprintf("  %s: %s (%d %s)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
		return
	}

	if latestRef == state.Release {
		cmdio.LogString(ctx, fmt.Sprintf("  %s: %s (%d %s, up to date)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
	} else {
		latestVersion := versionToken(installer.DisplaySkillsVersion(latestRef))
		cmdio.LogString(ctx, fmt.Sprintf("  %s: %s (%d %s)", label, version, len(state.Skills), skillNoun))
		cmdio.LogString(ctx, "  Update available: "+latestVersion)
		cmdio.LogString(ctx, "  Last updated: "+state.LastUpdated.Format("2006-01-02"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Run 'databricks aitools update' to update.")
	}
}
