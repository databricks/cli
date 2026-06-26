package aitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Package-level for testability. Tests override via update_test.go.
var updateSkillsFn = func(ctx context.Context, src installer.ManifestSource, installed []*agents.Agent, opts installer.UpdateOptions) (*installer.UpdateResult, error) {
	return installer.UpdateSkills(ctx, src, installed, opts)
}

var updatePluginsFn = installer.UpdateInstalledPlugins

func NewUpdateCmd() *cobra.Command {
	var check, force, noNew, noPrune bool
	var skillsFlag, scopeFlag string
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update installed skills and plugins",
		Long: `Update installed Databricks skills and plugins to the latest release.

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

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, true)
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

			// The plugin version recorded on update tracks the skills release.
			// Tolerate a resolution error (skip plugin updates, still do skills).
			ref, _, refErr := installer.GetSkillsRef(ctx)

			for _, scope := range scopes {
				if len(scopes) > 1 {
					cmdio.LogString(ctx, fmt.Sprintf("Updating %s skills...", scope))
				}

				dir := globalDir
				if scope == installer.ScopeProject {
					dir = projectDir
				}
				state, err := installer.LoadState(dir)
				if err != nil {
					return err
				}

				// Update plugin agents through their own CLI (check mode skips this).
				if !check && refErr == nil {
					pluginUpdates, err := updatePluginsFn(ctx, scope, ref)
					if err != nil {
						return err
					}
					for _, pu := range pluginUpdates {
						cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin v%s", pu.Agent, pu.Version))
					}
				}

				// Reconcile file skills for non-plugin agents. Skip entirely for a
				// pure-plugin install (no file skills to maintain); keep calling for
				// no-state installs so the "no skills or plugins installed" guidance still fires.
				if state == nil || len(state.Skills) > 0 {
					opts := installer.UpdateOptions{
						Check:   check,
						Force:   force,
						NoNew:   noNew,
						NoPrune: noPrune,
						Scope:   scope,
					}
					opts.Skills = skills

					result, err := updateSkillsFn(ctx, src, excludePluginAgents(installed, state), opts)
					if err != nil {
						return err
					}
					if result != nil && (len(result.Updated) > 0 || len(result.Added) > 0 || len(result.Removed) > 0) {
						cmdio.LogString(ctx, installer.FormatUpdateResult(result, check))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Show what would be updated without downloading")
	cmd.Flags().BoolVar(&force, "force", false, "Re-download even if versions match")
	cmd.Flags().BoolVar(&noNew, "no-new", false, "Don't auto-install new skills from manifest")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "Keep skills that vanished from the manifest instead of removing them")
	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to update (comma-separated)")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Update scope: project, global, or both")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Update project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Update globally-scoped skills")
	markScopeBoolsDeprecated(cmd)
	return cmd
}

// excludePluginAgents drops agents that are managed as plugins in this scope, so
// the file-skills reconcile never drops duplicate skill files onto a plugin agent.
func excludePluginAgents(installed []*agents.Agent, state *installer.InstallState) []*agents.Agent {
	if state == nil || len(state.Plugins) == 0 {
		return installed
	}
	out := make([]*agents.Agent, 0, len(installed))
	for _, a := range installed {
		if _, isPlugin := state.Plugins[a.Name]; !isPlugin {
			out = append(out, a)
		}
	}
	return out
}
