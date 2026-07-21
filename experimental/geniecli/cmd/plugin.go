package genieclicmd

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/log"
)

// pluginScope is the CLI install scope for the Databricks plugin. genie-cli is
// a global, workspace-wide agent, so it installs at global scope.
const pluginScope = installer.ScopeGlobal

// nativeScope is the agent-native plugin scope Codex installs into. Codex
// ignores --scope, but the installer records it, so "user" matches how a
// global CLI install maps to an agent's user-level scope.
const nativeScope = "user"

// ensureDatabricksPlugin installs the Databricks skills plugin for the agent if
// the CLI has not already recorded installing it. Codex must already be on PATH
// here (the configure step installs it) because the plugin installs through
// Codex's own CLI. Codex exposes no plugin manifest the CLI can read back, so
// prior installs are detected from the CLI's own install state rather than from
// the agent (unlike Claude Code).
func ensureDatabricksPlugin(ctx context.Context) error {
	agent := agents.ByName(agentName)
	if agent == nil {
		return fmt.Errorf("unknown agent %q", agentName)
	}

	installed, err := pluginRecorded(ctx, agent.Name)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	ref, _, err := installer.GetSkillsRef(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve the Databricks skills version: %w", err)
	}

	rec, err := installer.InstallPluginForAgent(ctx, agent, nativeScope, ref)
	if err != nil {
		return fmt.Errorf("failed to install the Databricks skills plugin: %w", err)
	}

	// Record the install so the refresh path (and `databricks aitools`) can act
	// on exactly where we installed.
	records := map[string]installer.PluginRecord{agent.Name: rec}
	if err := installer.RecordPluginInstalls(ctx, pluginScope, records, ref); err != nil {
		return fmt.Errorf("failed to record the Databricks skills plugin install: %w", err)
	}
	return nil
}

// pluginRecorded reports whether the CLI's install state already records the
// Databricks plugin for the named agent at the global scope.
func pluginRecorded(ctx context.Context, name string) (bool, error) {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return false, err
	}
	state, err := installer.LoadState(dir)
	if err != nil {
		return false, fmt.Errorf("failed to read the Databricks skills install state: %w", err)
	}
	if state == nil {
		return false, nil
	}
	_, ok := state.Plugins[name]
	return ok, nil
}

// refreshDatabricksPlugin updates the Databricks skills plugin to the latest
// release before the agent starts. Failures are non-fatal: a stale plugin still
// works, so an offline or unreachable-GitHub run warns and launches anyway.
func refreshDatabricksPlugin(ctx context.Context) {
	ref, _, err := installer.GetSkillsRef(ctx)
	if err != nil {
		log.Warnf(ctx, "Skipping Databricks skills update: %v", err)
		return
	}
	if _, err := installer.UpdateInstalledPlugins(ctx, pluginScope, ref); err != nil {
		log.Warnf(ctx, "Skipping Databricks skills update: %v", err)
	}
}
