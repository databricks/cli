package installer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// UninstallOptions controls the behavior of UninstallSkillsWithResult.
type UninstallOptions struct {
	Skills          []string // empty = all
	Agents          []string // empty = all agents
	Scope           string   // ScopeGlobal or ScopeProject (default: global)
	KeepMarketplace bool     // keep the marketplace registration when removing a plugin
}

// removeAllFn removes a skill's canonical dir or a targeted agent's entry. A
// seam for tests.
var removeAllFn = os.RemoveAll

// UninstallResult reports what an uninstall did. A plugin or skill that could
// not be removed is only warned about, so the run still returns no error;
// telemetry reads PluginFailures and SkillFailures to tell a run that removed
// nothing from a success.
type UninstallResult struct {
	PluginsRemoved int
	// SkillsChanged counts skills of which this run removed at least one copy an
	// agent reads: an agent's symlink or entry, or the canonical dir. Without
	// opts.Agents, a skill counts when its canonical dir is removed or was
	// already gone. With opts.Agents set, removing only the canonical dir does
	// not count when a targeted agent's entry could not be removed, since that
	// agent keeps the skill.
	SkillsChanged  int
	PluginFailures []error
	// SkillFailures holds the errors from removing a skill's canonical dir or
	// a targeted agent's entry.
	SkillFailures []error
}

// UninstallSkillsWithResult removes installed skills based on options and
// reports what was removed and what could not be.
// When opts.Skills is empty, all skills (and plugins) are removed.
// When opts.Skills is non-empty, only the named skills are removed.
func UninstallSkillsWithResult(ctx context.Context, opts UninstallOptions) (*UninstallResult, error) {
	scope := opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}

	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return nil, err
	}

	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
	}

	state, err := LoadState(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load install state: %w", err)
	}

	if !state.HasSkillsOrPlugins() {
		if scope == ScopeGlobal && hasLegacyInstall(ctx, baseDir) {
			return nil, fmt.Errorf("%w; run 'databricks aitools install' first, then uninstall", ErrLegacyInstall)
		}
		return nil, ErrNothingInstalled
	}

	// A full uninstall (no --skills filter) also tears down plugins.
	fullUninstall := len(opts.Skills) == 0
	agentFilter := stringSet(opts.Agents)
	targetedAgents := len(agentFilter) > 0

	pluginCount := 0
	var pluginFailures []error
	if fullUninstall {
		for _, name := range slices.Sorted(maps.Keys(state.Plugins)) {
			if targetedAgents && !agentFilter[name] {
				continue
			}
			agent := agents.ByName(name)
			if agent == nil {
				// We can't tear down an agent we don't know; keep the record (and
				// the state file) rather than silently dropping a live plugin.
				log.Warnf(ctx, "Leaving plugin record for unknown agent %q; remove its plugin manually", name)
				pluginFailures = append(pluginFailures, unknownAgentInStateError(name))
				continue
			}
			rec := state.Plugins[name]
			if err := UninstallPluginForAgent(ctx, agent, rec, opts.KeepMarketplace); err != nil {
				log.Warnf(ctx, "Skipped %s: %v", agent.DisplayName, err)
				pluginFailures = append(pluginFailures, err)
				continue
			}
			delete(state.Plugins, name)
			pluginCount++
			// Mirror the de-register condition in UninstallPluginForAgent: a Shared
			// marketplace is never removed, so don't claim we did.
			note := " + marketplace"
			if opts.KeepMarketplace || !rec.InstalledMarketplace || agent.Plugin.Shared {
				note = ""
			}
			cmdio.LogString(ctx, fmt.Sprintf("  %s  removed databricks plugin%s", agent.DisplayName, note))
		}
	}

	// Determine which skills to remove.
	var toRemove []string
	if len(opts.Skills) > 0 {
		seen := make(map[string]bool)
		for _, name := range opts.Skills {
			if _, ok := state.Skills[name]; !ok {
				if targetedAgents {
					if foundIn, found := untrackedSkillAgents(ctx, name, opts.Agents, scope, cwd); found {
						return nil, fmt.Errorf("skill %q is not tracked in Databricks aitools state but exists for %s. Run 'databricks aitools install --skills-only --agents %s' to rebuild state before uninstalling, or remove it manually", name, strings.Join(foundIn, ", "), strings.Join(opts.Agents, ","))
					}
				}
				return nil, &SkillError{Skill: name, Reason: ReasonSkillNotFound, Detail: "is not installed"}
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			toRemove = append(toRemove, name)
		}
	} else {
		// Sorted, like plugins above, so SkillFailures has a stable order.
		toRemove = slices.Sorted(maps.Keys(state.Skills))
	}

	// Remove skill directories and symlinks for each skill. skillsChanged counts
	// the skills of which a copy was removed (see UninstallResult.SkillsChanged);
	// skillsUninstalled also counts skills whose state record is dropped
	// without a failure though no copy was left to remove, e.g. a stale entry.
	skillsChanged, skillsUninstalled := 0, 0
	var skillFailures []error
	for _, name := range toRemove {
		canonicalDir := filepath.Join(baseDir, name)
		removeCanonical := true
		changed := false
		var failures []error
		if targetedAgents {
			changed, failures = removeSkillExposuresFromAgents(ctx, name, canonicalDir, scope, cwd, state, opts.Agents)
			removeCanonical = !hasRemainingSkillExposure(ctx, name, canonicalDir, scope, cwd, opts.Agents)
		} else {
			changed = removeSymlinksFromAgents(ctx, name, canonicalDir, scope, cwd)
		}
		// A targeted agent that keeps its entry still has the skill, so removing
		// the canonical dir alone is no change for it.
		exposureFailed := len(failures) > 0
		if removeCanonical {
			// RemoveAll succeeds on a missing dir. Without --agents the skill's
			// state record is dropped either way, so it counts; with --agents
			// only an existing dir does.
			_, statErr := os.Lstat(canonicalDir)
			if err := removeAllFn(canonicalDir); err != nil {
				log.Warnf(ctx, "Failed to remove %s: %v", canonicalDir, err)
				failures = append(failures, err)
			} else if (statErr == nil || !targetedAgents) && !exposureFailed {
				changed = true
			}
		}
		skillFailures = append(skillFailures, failures...)
		if changed {
			skillsChanged++
		}
		// A skill with no copy left to remove still counts when its state
		// record is dropped without a failure (e.g. a stale entry).
		if changed || (removeCanonical && len(failures) == 0) {
			skillsUninstalled++
		}
		if !removeCanonical {
			continue
		}
		delete(state.Skills, name)
		delete(state.RepoDirs, name)
		for path := range state.Files {
			if strings.HasPrefix(path, name+"/") {
				delete(state.Files, path)
			}
		}
	}

	// If nothing remains in this scope, clean up orphaned symlinks. Delete the
	// state file unless it still holds MCP records, which uninstall does not own;
	// otherwise persist the trimmed state. The MCP records are read from disk now,
	// since an MCP save may have happened after state was loaded above.
	noSkillsOrPlugins := len(state.Skills) == 0 && len(state.Plugins) == 0
	if noSkillsOrPlugins {
		cleanOrphanedSymlinks(ctx, baseDir, scope, cwd)
	}
	mcpRecorded, err := mcpRecordedOnDisk(baseDir)
	if err != nil {
		return err
	}
	if noSkillsOrPlugins && !mcpRecorded {
		stateFile := filepath.Join(baseDir, stateFileName)
		if err := os.Remove(stateFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("failed to remove state file: %w", err)
		}
	} else {
		if err := SaveState(baseDir, state); err != nil {
			return nil, err
		}
	}

	if pluginCount > 0 {
		noun := "agent"
		if pluginCount != 1 {
			noun = "agents"
		}
		cmdio.LogString(ctx, fmt.Sprintf("Uninstalled the plugin from %d %s.", pluginCount, noun))
	}
	if skillsUninstalled > 0 {
		noun := "skills"
		if skillsUninstalled == 1 {
			noun = "skill"
		}
		cmdio.LogString(ctx, fmt.Sprintf("Uninstalled %d %s.", skillsUninstalled, noun))
	}
	return &UninstallResult{PluginsRemoved: pluginCount, SkillsChanged: skillsChanged, PluginFailures: pluginFailures, SkillFailures: skillFailures}, nil
}

func stringSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// removeSkillExposuresFromAgents removes the skill from the named agents' skills
// dirs and reports whether it removed an existing entry for any of them, plus
// the errors from entries it failed to remove.
func removeSkillExposuresFromAgents(ctx context.Context, skillName, canonicalDir, scope, cwd string, state *InstallState, agentNames []string) (bool, []error) {
	removed := false
	var failures []error
	for _, name := range agentNames {
		agent := agents.ByName(name)
		if agent == nil {
			continue
		}
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}
		entry := filepath.Join(agentDir, skillName)
		if !exposureRemovable(entry, canonicalDir, skillName, state) {
			log.Debugf(ctx, "Skipping unmanaged skill %s for %s", entry, agent.DisplayName)
			continue
		}
		// exposureRemovable also accepts a missing entry; there is nothing to remove.
		if _, err := os.Lstat(entry); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := removeAllFn(entry); err != nil {
			log.Warnf(ctx, "Failed to remove %s: %v", entry, err)
			failures = append(failures, err)
			continue
		}
		removed = true
	}
	return removed, failures
}

func hasRemainingSkillExposure(ctx context.Context, skillName, canonicalDir, scope, cwd string, removedAgents []string) bool {
	removed := stringSet(removedAgents)
	for _, agent := range agents.Registry {
		if removed[agent.Name] {
			continue
		}
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}
		entry := filepath.Join(agentDir, skillName)
		if skillExposureExists(entry, canonicalDir) {
			return true
		}
	}
	return false
}

func skillExposureExists(entry, canonicalDir string) bool {
	fi, err := os.Lstat(entry)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return true
	}
	target, err := os.Readlink(entry)
	if err != nil {
		return true
	}
	abs := target
	if !filepath.IsAbs(target) {
		abs = filepath.Clean(filepath.Join(filepath.Dir(entry), target))
	}
	return abs == canonicalDir || strings.HasPrefix(abs, canonicalDir+string(os.PathSeparator))
}

func untrackedSkillAgents(ctx context.Context, skillName string, agentNames []string, scope, cwd string) ([]string, bool) {
	var found []string
	for _, name := range agentNames {
		agent := agents.ByName(name)
		if agent == nil {
			continue
		}
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}
		if _, err := os.Lstat(filepath.Join(agentDir, skillName)); err == nil {
			found = append(found, agent.DisplayName)
		}
	}
	return found, len(found) > 0
}

// removeSymlinksFromAgents removes a skill's symlink from all agent directories
// in the registry, but only if the entry is a symlink pointing into canonicalDir.
// Non-symlink directories are left untouched to avoid deleting user-managed content.
// It reports whether it removed any symlink.
func removeSymlinksFromAgents(ctx context.Context, skillName, canonicalDir, scope, cwd string) bool {
	removed := false
	for _, agent := range agents.Registry {
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}

		destDir := filepath.Join(agentDir, skillName)

		// Use Lstat to detect symlinks (Stat follows them).
		fi, err := os.Lstat(destDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			log.Warnf(ctx, "Failed to stat %s for %s: %v", destDir, agent.DisplayName, err)
			continue
		}

		if fi.Mode()&os.ModeSymlink == 0 {
			log.Debugf(ctx, "Skipping non-symlink %s for %s", destDir, agent.DisplayName)
			continue
		}

		target, err := os.Readlink(destDir)
		if err != nil {
			log.Warnf(ctx, "Failed to read symlink %s: %v", destDir, err)
			continue
		}

		// Resolve relative symlinks to absolute for comparison.
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(destDir), target)
			absTarget = filepath.Clean(absTarget)
		}

		// Only remove if the symlink points into our canonical dir.
		if !strings.HasPrefix(absTarget, canonicalDir+string(os.PathSeparator)) && absTarget != canonicalDir {
			log.Debugf(ctx, "Skipping symlink %s (points to %s, not %s)", destDir, absTarget, canonicalDir)
			continue
		}

		if err := os.Remove(destDir); err != nil {
			log.Warnf(ctx, "Failed to remove symlink %s: %v", destDir, err)
		} else {
			removed = true
			log.Debugf(ctx, "Removed %q from %s", skillName, agent.DisplayName)
		}
	}
	return removed
}

// cleanOrphanedSymlinks scans all agent skill directories for symlinks pointing
// into baseDir that are not tracked in state, and removes them.
func cleanOrphanedSymlinks(ctx context.Context, baseDir, scope, cwd string) {
	for _, agent := range agents.Registry {
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}

		entries, err := os.ReadDir(agentDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			entryPath := filepath.Join(agentDir, entry.Name())

			fi, err := os.Lstat(entryPath)
			if err != nil {
				continue
			}

			if fi.Mode()&os.ModeSymlink == 0 {
				continue
			}

			target, err := os.Readlink(entryPath)
			if err != nil {
				continue
			}

			// Resolve relative symlinks to absolute for comparison.
			absTarget := target
			if !filepath.IsAbs(target) {
				absTarget = filepath.Clean(filepath.Join(filepath.Dir(entryPath), target))
			}

			// Check if the symlink points into our managed skills dir.
			if !strings.HasPrefix(absTarget, baseDir+string(os.PathSeparator)) && absTarget != baseDir {
				continue
			}

			// This symlink points into our managed dir. Remove it.
			if err := os.Remove(entryPath); err != nil {
				log.Warnf(ctx, "Failed to remove orphaned symlink %s: %v", entryPath, err)
			} else {
				log.Debugf(ctx, "Removed orphaned symlink %s -> %s", entryPath, target)
			}
		}
	}
}
