package aitools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// Package-level for testability. Tests override via uninstall_test.go.
var uninstallSkillsFn = installer.UninstallSkillsWithResult

func NewUninstallCmd() *cobra.Command {
	var skillsFlag, agentsFlag, scopeFlag string
	var projectFlag, globalFlag, keepMarketplace bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Databricks skills and plugins",
		Long: `Remove installed Databricks skills and plugins from all coding agents.

By default, removes all skills. Use --skills to remove specific skills only.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, false)
			if err != nil {
				return withCategory(err, protos.AitoolsErrorCategoryInvalidFlags)
			}
			// Record a scope flag before resolving, so a scope error still reports it.
			funnel := funnelFrom(ctx)
			funnel.setScope(explicitScope(projectFlag, globalFlag))

			globalDir, err := installer.GlobalSkillsDir(ctx)
			if err != nil {
				return err
			}
			projectDir, err := installer.ProjectSkillsDir(ctx)
			if err != nil {
				return err
			}

			scope, err := resolveScopeForUninstall(ctx, projectFlag, globalFlag, globalDir, projectDir)
			if err != nil {
				return err
			}
			funnel.setScope(scope)

			opts := installer.UninstallOptions{
				Scope:           scope,
				KeepMarketplace: keepMarketplace,
			}
			opts.Skills = splitAndTrim(skillsFlag)
			if agentsFlag != "" {
				targetAgents, err := resolveAgentNames(ctx, agentsFlag)
				if err != nil {
					return err
				}
				funnel.setTargets(targetAgents)
				opts.Agents = make([]string, 0, len(targetAgents))
				for _, agent := range targetAgents {
					opts.Agents = append(opts.Agents, agent.Name)
				}
			}

			// Uninstall is destructive, so confirm interactively before doing
			// anything. Non-interactive runs (no TTY) proceed unprompted so
			// automation is unaffected.
			dir := globalDir
			if scope == installer.ScopeProject {
				dir = projectDir
			}
			state, err := installer.LoadState(dir)
			if err != nil {
				return fmt.Errorf("failed to load install state: %w", err)
			}
			if cmdio.IsPromptSupported(ctx) {
				proceed, err := confirmUninstall(ctx, state, opts)
				if err != nil {
					return err
				}
				if !proceed {
					cmdio.LogString(ctx, "Cancelled.")
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeCancelled, protos.AitoolsErrorCategoryUnspecified)
					return nil
				}
			}

			// With something installed the run removes files and runs agent CLIs,
			// so it waits for the telemetry auth. The flag errors and the declined
			// confirm above do not, nor does a run that only reports that nothing
			// (or a legacy install) is there, or that --skills names a skill that
			// is not installed (the installer returns that error).
			if state != nil && skillsInstalled(state, opts.Skills) {
				waitForTelemetryAuth(ctx)
			}
			result, err := uninstallSkillsFn(ctx, opts)
			if err != nil {
				return err
			}
			// A plugin or skill that could not be removed is only a warning.
			failures := slices.Concat(result.PluginFailures, result.SkillFailures)
			if result.PluginsRemoved == 0 && result.SkillsRemoved == 0 {
				if len(failures) > 0 {
					// Every removal failed: FAILURE, though it exits 0.
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeFailure, classifyInstallError(failures[0]))
				} else {
					// E.g. --agents names agents that have none of the skills.
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeNothingToDo, protos.AitoolsErrorCategoryUnspecified)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to uninstall (comma-separated)")
	cmd.Flags().StringVar(&agentsFlag, "agents", "", "Agents to uninstall from (comma-separated, e.g. claude-code,cursor)")
	cmd.Flags().BoolVar(&keepMarketplace, "keep-marketplace", false, "Keep the marketplace registration when removing a plugin")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Uninstall scope: project or global")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Uninstall project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Uninstall globally-scoped skills")
	markScopeBoolsDeprecated(cmd)
	return instrumentCommand(cmd, protos.AitoolsCommandUninstall)
}

// confirmUninstall asks the user to confirm a destructive uninstall, summarizing
// what will be removed. It returns true without prompting when nothing is
// recorded in the scope (state is nil), so the installer can surface its own
// "nothing installed" or legacy guidance instead.
func confirmUninstall(ctx context.Context, state *installer.InstallState, opts installer.UninstallOptions) (bool, error) {
	msg, ask := uninstallConfirmMessage(state, opts)
	if !ask {
		return true, nil
	}
	cmdio.LogString(ctx, msg)
	proceed, err := cmdio.AskYesOrNo(ctx, "Proceed?")
	if err != nil {
		return false, promptError(err)
	}
	return proceed, nil
}

// uninstallConfirmMessage builds the human summary of what an uninstall will
// remove. ask is false when there is nothing recorded to confirm.
func uninstallConfirmMessage(state *installer.InstallState, opts installer.UninstallOptions) (msg string, ask bool) {
	if state == nil {
		return "", false
	}
	target := "all agents"
	if len(opts.Agents) > 0 {
		target = strings.Join(opts.Agents, ", ")
	}
	if len(opts.Skills) > 0 {
		return fmt.Sprintf("This will remove %s %s from %s (%s scope).", plural(len(opts.Skills), "skill"), strings.Join(opts.Skills, ", "), target, opts.Scope), true
	}
	var parts []string
	if n := len(state.Skills); n > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", n, plural(n, "skill")))
	}
	if n := len(state.Plugins); n > 0 {
		parts = append(parts, fmt.Sprintf("the databricks plugin from %d %s", n, plural(n, "agent")))
	}
	if len(parts) == 0 {
		return "", false
	}
	return fmt.Sprintf("This will remove %s from %s (%s scope).", strings.Join(parts, " and "), target, opts.Scope), true
}

// skillsInstalled reports whether every name is recorded in state.
func skillsInstalled(state *installer.InstallState, names []string) bool {
	for _, name := range names {
		if _, ok := state.Skills[name]; !ok {
			return false
		}
	}
	return true
}

// plural returns noun for n == 1 and noun+"s" otherwise.
func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}
