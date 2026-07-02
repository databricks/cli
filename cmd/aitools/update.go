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
)

// Package-level for testability. Tests override via update_test.go.
var updateSkillsFn = func(ctx context.Context, src installer.ManifestSource, installed []*agents.Agent, opts installer.UpdateOptions) (*installer.UpdateResult, error) {
	return installer.UpdateSkills(ctx, src, installed, opts)
}

var (
	updatePluginsFn               = installer.UpdateInstalledPlugins
	updateInstallPluginForAgentFn = installer.InstallPluginForAgent
	updateRecordPluginInstallsFn  = installer.RecordPluginInstalls
	updateCleanupLegacyFn         = installer.RemoveLegacyRawSkills
	hasManagedRawSkillsForAgentFn = installer.HasManagedRawSkillsForAgent
)

type pluginMigrationCandidate struct {
	agent       *agents.Agent
	nativeScope string
}

type pluginMigrationResult struct {
	agent  *agents.Agent
	record installer.PluginRecord
}

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

				if check && refErr == nil && state != nil && len(state.Plugins) > 0 {
					printPluginCheckResults(ctx, state, installer.DisplaySkillsVersion(ref))
				}

				migrationCandidates, err := pluginMigrationCandidatesForScope(ctx, scope, state)
				if err != nil {
					return err
				}
				if check {
					printPluginMigrationCheckResults(ctx, migrationCandidates)
				}

				// Update plugin agents through their own CLI (check mode skips this).
				if !check && refErr == nil {
					pluginUpdates, err := updatePluginsFn(ctx, scope, ref)
					if err != nil {
						return err
					}
					for _, pu := range pluginUpdates {
						cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s", pu.Agent, versionToken(pu.Version)))
					}

					migrated, err := migrateLegacyRawSkillsToPlugins(ctx, scope, ref, migrationCandidates)
					if err != nil {
						return err
					}
					if migrated {
						state, err = installer.LoadState(dir)
						if err != nil {
							return err
						}
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
					if result != nil && (check || len(result.Updated) > 0 || len(result.Added) > 0 || len(result.Removed) > 0) {
						text := installer.FormatUpdateResult(result, check)
						if text != "No changes." || len(migrationCandidates) == 0 {
							cmdio.LogString(ctx, text)
						}
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

func pluginMigrationCandidatesForScope(ctx context.Context, scope string, state *installer.InstallState) ([]pluginMigrationCandidate, error) {
	var candidates []pluginMigrationCandidate
	for _, agent := range agents.Registry {
		if agent.Plugin == nil {
			continue
		}
		if state != nil {
			if _, alreadyPlugin := state.Plugins[agent.Name]; alreadyPlugin {
				continue
			}
		}
		nativeScope, ok, _ := mapAgentScope(agent, scope)
		if !ok {
			continue
		}
		// Match install's non-interactive behavior: a raw install for a plugin
		// agent should attempt migration and report a skip if its CLI is missing,
		// rather than silently staying on raw skills forever.
		hasRawSkills, err := hasManagedRawSkillsForAgentFn(ctx, agent, scope)
		if err != nil {
			return nil, err
		}
		if !hasRawSkills {
			continue
		}
		candidates = append(candidates, pluginMigrationCandidate{agent: agent, nativeScope: nativeScope})
	}
	return candidates, nil
}

func printPluginMigrationCheckResults(ctx context.Context, candidates []pluginMigrationCandidate) {
	for _, candidate := range candidates {
		cmdio.LogString(ctx, fmt.Sprintf("  %s  would migrate raw skills to databricks plugin", candidate.agent.DisplayName))
	}
}

func migrateLegacyRawSkillsToPlugins(ctx context.Context, scope, ref string, candidates []pluginMigrationCandidate) (bool, error) {
	if len(candidates) == 0 {
		return false, nil
	}

	records := map[string]installer.PluginRecord{}
	migrated := make([]pluginMigrationResult, 0, len(candidates))
	for _, candidate := range candidates {
		agent := candidate.agent
		cmdio.LogString(ctx, fmt.Sprintf("Installing databricks plugin for %s...", agent.DisplayName))
		rec, err := updateInstallPluginForAgentFn(ctx, agent, candidate.nativeScope, ref)
		if err != nil {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, fmt.Sprintf("Skipped %s: %v", agent.DisplayName, err)))
			continue
		}
		records[agent.Name] = rec
		migrated = append(migrated, pluginMigrationResult{agent: agent, record: rec})
	}
	if len(records) == 0 {
		return false, nil
	}
	if err := updateRecordPluginInstallsFn(ctx, scope, records, ref); err != nil {
		return false, err
	}
	for _, result := range migrated {
		if err := updateCleanupLegacyFn(ctx, result.agent, scope); err != nil {
			log.Debugf(ctx, "Legacy skill cleanup for %s failed: %v", result.agent.DisplayName, err)
		}
		cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s", result.agent.DisplayName, versionToken(result.record.Version)))
	}
	return true, nil
}

func printPluginCheckResults(ctx context.Context, state *installer.InstallState, latest string) {
	for _, name := range slices.Sorted(maps.Keys(state.Plugins)) {
		rec := state.Plugins[name]
		status := "up to date"
		if rec.Version == "" {
			status = "version unknown"
		} else if rec.Version != latest {
			status = "update available"
		}
		cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s (%s)", agentDisplayName(name), versionToken(rec.Version), status))
	}
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
