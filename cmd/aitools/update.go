package aitools

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// Package-level for testability. Tests override via update_test.go.
var updateSkillsFn = func(ctx context.Context, src installer.ManifestSource, installed []*agents.Agent, opts installer.UpdateOptions) (*installer.UpdateResult, error) {
	return installer.UpdateSkills(ctx, src, installed, opts)
}

var (
	updateGetSkillsRefFn          = installer.GetSkillsRef
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
preview what would change without downloading.

The command exits with code 1 when every update or check it attempts fails.`,
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
				return withCategory(err, protos.AitoolsErrorCategoryInvalidFlags)
			}
			// Record a scope flag before resolving, so a scope error still reports it.
			funnel := funnelFrom(ctx)
			funnel.setScope(explicitScope(projectFlag, globalFlag))

			scopes, err := resolveScopeForUpdate(ctx, projectFlag, globalFlag, globalDir, projectDir)
			if err != nil {
				return err
			}
			// The scope enum has no "both"; a run over both scopes reports none.
			if len(scopes) == 1 {
				funnel.setScope(scopes[0])
			}

			installed := agents.DetectInstalled(ctx)
			src := &installer.GitHubManifestSource{}
			skills := splitAndTrim(skillsFlag)

			// The plugin version recorded on update tracks the skills release.
			// Tolerate a resolution error (skip plugin updates, still do skills).
			ref, _, refErr := updateGetSkillsRefFn(ctx)

			// A failed or skipped plugin update, or a check whose manifest could
			// not be fetched, is only a warning, unless every attempt failed and
			// nothing else ran. A plugin record for an agent this CLI does not know
			// is not an attempt: no command can act on it.
			// firstErr is the first failure of any kind; firstAttemptErr the first
			// one of an attempt (a record of an unknown agent is not one).
			var firstErr, firstAttemptErr error
			updatedAnything := false

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
				// A scope with something installed fetches releases and runs agent
				// CLIs, so the run waits for the telemetry auth. Without state the
				// run only reports that nothing (or a legacy install) is there.
				if state != nil {
					waitForTelemetryAuth(ctx)
				}

				if check && refErr == nil && state != nil && len(state.Plugins) > 0 {
					printPluginCheckResults(ctx, state, installer.DisplaySkillsVersion(ref))
					updatedAnything = true
				}

				migrationCandidates, err := pluginMigrationCandidatesForScope(ctx, scope, state)
				if err != nil {
					return err
				}
				if check {
					printPluginMigrationCheckResults(ctx, migrationCandidates)
				}

				// Without a ref the recorded plugins are skipped (with --check,
				// not checked).
				if refErr != nil && state != nil && hasKnownPluginRecord(state) {
					firstAttemptErr = cmp.Or(firstAttemptErr, refErr)
					firstErr = cmp.Or(firstErr, refErr)
				}

				// Update plugin agents through their own CLI (check mode skips this).
				if !check && refErr == nil {
					pluginUpdates, err := updatePluginsFn(ctx, scope, ref)
					if err != nil {
						return err
					}
					for _, pu := range pluginUpdates {
						if pu.Err != nil {
							if !errors.Is(pu.Err, installer.ErrUnknownAgent) {
								firstAttemptErr = cmp.Or(firstAttemptErr, pu.Err)
							}
							firstErr = cmp.Or(firstErr, pu.Err)
							continue
						}
						updatedAnything = true
						cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s", pu.Agent, versionToken(pu.Version)))
					}

					migrated, err := migrateLegacyRawSkillsToPlugins(ctx, scope, ref, migrationCandidates)
					if err != nil {
						return err
					}
					if migrated {
						updatedAnything = true
						state, err = installer.LoadState(dir)
						if err != nil {
							return err
						}
					}
				}

				// Reconcile file skills for non-plugin agents. Skip entirely for a
				// pure-plugin install (no file skills to maintain); keep calling for
				// no-state installs so the "no skills or plugins installed" guidance still fires.
				if !state.HasSkillsOrPlugins() || len(state.Skills) > 0 {
					opts := installer.UpdateOptions{
						Check:   check,
						Force:   force,
						NoNew:   noNew,
						NoPrune: noPrune,
						Scope:   scope,
					}
					opts.Skills = skills

					targetAgents := installed
					if scope == installer.ScopeProject {
						cwd, err := os.Getwd()
						if err != nil {
							return err
						}
						targetAgents = mergeAgents(installed, agents.DetectProjectInstalled(cwd))
					}

					fileAgents, manifestPlugins := excludePluginAgents(ctx, scope, targetAgents, state)
					// Raw skills an earlier install gave an agent that now has the
					// plugin would show every skill twice, so remove them as
					// install does when it keeps the plugin.
					if !check {
						for _, a := range manifestPlugins {
							if err := updateCleanupLegacyFn(ctx, a, scope); err != nil {
								log.Debugf(ctx, "Legacy skill cleanup for %s failed: %v", a.DisplayName, err)
							}
						}
					}
					// The plugin agents left no agent that can take file skills in
					// this scope, so there is nothing to reconcile. Without this, a
					// project whose only project-capable agent has the plugin would
					// fail the update with "no agents support project-scoped skills".
					// Without install state the reconcile still runs, for the
					// "no skills or plugins installed" and legacy-install guidance.
					if state != nil && len(manifestPlugins) > 0 && !slices.ContainsFunc(fileAgents, func(a *agents.Agent) bool {
						return scope != installer.ScopeProject || a.SupportsProjectScope
					}) {
						continue
					}

					result, err := updateSkillsFn(ctx, src, fileAgents, opts)
					if err != nil {
						return err
					}
					if result != nil && result.CheckErr != nil {
						firstAttemptErr = cmp.Or(firstAttemptErr, result.CheckErr)
						firstErr = cmp.Or(firstErr, result.CheckErr)
					} else {
						updatedAnything = true
					}
					// A check that could not fetch the manifest has nothing to report.
					if result != nil && result.CheckErr == nil && (check || len(result.Updated) > 0 || len(result.Added) > 0 || len(result.Removed) > 0) {
						text := installer.FormatUpdateResult(result, check)
						if text != "No changes." || len(migrationCandidates) == 0 {
							cmdio.LogString(ctx, text)
						}
					}
				}
			}
			if firstErr != nil && !updatedAnything {
				if firstAttemptErr == nil {
					// Only unknown-agent records: warned about above, but not a run
					// that failed to do what it could.
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeFailure, classifyInstallError(firstErr))
					return nil
				}
				if check {
					return fmt.Errorf("nothing was checked: %w", firstAttemptErr)
				}
				return fmt.Errorf("nothing was updated: %w", firstAttemptErr)
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
	return instrumentCommand(cmd, protos.AitoolsCommandUpdate)
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
		// Match install's behavior: an agent whose CLI is not on PATH keeps its
		// raw skills, which update refreshes, until the CLI is there to install
		// the plugin with. An agent with the CLI attempts migration and reports
		// a skip if it fails, rather than silently staying on raw skills forever.
		if !agent.HasBinary(ctx) {
			continue
		}
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
			status = "update may be available"
		} else if rec.Version != latest {
			status = "update available"
		}
		cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s (%s)", agentDisplayName(name), versionToken(rec.Version), status))
	}
}

// mergeAgents concatenates two agent lists, skipping duplicate names.
func mergeAgents(installed, project []*agents.Agent) []*agents.Agent {
	seen := make(map[string]bool, len(installed))
	merged := make([]*agents.Agent, 0, len(installed)+len(project))
	for _, a := range installed {
		seen[a.Name] = true
		merged = append(merged, a)
	}
	for _, a := range project {
		if !seen[a.Name] {
			merged = append(merged, a)
		}
	}
	return merged
}

// excludePluginAgents splits off agents that have the plugin in this scope, so
// the file-skills reconcile never drops duplicate skill files onto a plugin
// agent. That is an agent the CLI recorded a plugin install for, or one whose
// own manifest shows a plugin serving the scope (see manifestPluginActive) while
// its CLI is missing: install keeps such a plugin, without recording it. With
// the CLI on PATH, raw skills next to the plugin are the user's choice (install
// --skills-only), so they are refreshed as before. manifestPlugins lists the
// agents excluded for their manifest.
func excludePluginAgents(ctx context.Context, scope string, installed []*agents.Agent, state *installer.InstallState) (fileAgents, manifestPlugins []*agents.Agent) {
	fileAgents = make([]*agents.Agent, 0, len(installed))
	for _, a := range installed {
		if state != nil {
			if _, isPlugin := state.Plugins[a.Name]; isPlugin {
				continue
			}
		}
		if !a.HasBinary(ctx) {
			if active, err := manifestPluginActive(ctx, a, scope); err == nil && active {
				manifestPlugins = append(manifestPlugins, a)
				continue
			}
		}
		fileAgents = append(fileAgents, a)
	}
	return fileAgents, manifestPlugins
}

// hasKnownPluginRecord reports whether state records a plugin for an agent this
// CLI knows, so that skipping the recorded plugins is a failed attempt.
func hasKnownPluginRecord(state *installer.InstallState) bool {
	for name := range state.Plugins {
		if agents.ByName(name) != nil {
			return true
		}
	}
	return false
}
