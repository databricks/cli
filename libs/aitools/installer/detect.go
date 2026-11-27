package installer

import (
	"context"
	"slices"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/log"
)

// InstalledTools returns "<agent>_<version>" tokens for every coding tool that
// has the Databricks plugin installed, sorted. The version is taken from the
// agent's own plugin cache (ground truth, catches installs made directly through
// the agent), falling back to the version recorded in the CLI install state when
// the plugin is not found on disk. An agent with no resolvable version is
// omitted rather than reported without one. That includes an agent without a
// plugin cache reader (Codex) whose record has no version because its last
// update ran from a marketplace copy that could not be refreshed; use
// InstalledPluginAgents to ask whether a plugin is installed at all.
func InstalledTools(ctx context.Context) []string {
	recorded := recordedPluginVersions(ctx)

	var tools []string
	for _, a := range agents.Registry {
		if a.Plugin == nil {
			continue
		}
		version, _ := a.DatabricksPluginVersion(ctx)
		if version == "" {
			version = recorded[a.Name]
		}
		if version != "" {
			tools = append(tools, a.Name+"_"+version)
		}
	}
	slices.Sort(tools)
	return tools
}

// InstalledPluginAgents returns the names of the coding tools that have the
// Databricks plugin installed, sorted: found in the agent's own plugin cache or
// recorded in the CLI install state, whether or not a version is known.
func InstalledPluginAgents(ctx context.Context) []string {
	recorded := recordedPluginVersions(ctx)

	var names []string
	for _, a := range agents.Registry {
		if a.Plugin == nil {
			continue
		}
		_, inCache := a.DatabricksPluginVersion(ctx)
		_, inState := recorded[a.Name]
		if inCache || inState {
			names = append(names, a.Name)
		}
	}
	slices.Sort(names)
	return names
}

// recordedPluginVersions maps agent name to the databricks plugin version
// recorded in the CLI install state, merged across global and project scope.
// An agent whose record has no version maps to "". When both scopes record a
// version, the project scope wins (it is the more specific,
// closer-to-the-invocation install); a scope without a version never replaces
// one with a version.
func recordedPluginVersions(ctx context.Context) map[string]string {
	versions := map[string]string{}
	// Load global first, then project, so project overwrites on conflict.
	for _, scope := range []string{ScopeGlobal, ScopeProject} {
		dir, err := skillsDir(ctx, scope)
		if err != nil {
			continue
		}
		state, err := LoadState(dir)
		if err != nil {
			log.Debugf(ctx, "Could not load AI Tools install state in %s: %v", dir, err)
			continue
		}
		if state == nil {
			continue
		}
		for name, rec := range state.Plugins {
			if _, seen := versions[name]; !seen || rec.Version != "" {
				versions[name] = rec.Version
			}
		}
	}
	return versions
}
