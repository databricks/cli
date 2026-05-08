package installer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// UninstallOptions controls the behavior of UninstallSkillsOpts.
type UninstallOptions struct {
	Skills []string // empty = all
	Scope  string   // ScopeGlobal or ScopeProject (default: global)
}

// UninstallSkills removes all installed skills, their symlinks, and the state file.
func UninstallSkills(ctx context.Context) error {
	return UninstallSkillsOpts(ctx, UninstallOptions{})
}

// UninstallSkillsOpts removes installed skills based on options.
// When opts.Skills is empty, all skills are removed (same as UninstallSkills).
// When opts.Skills is non-empty, only the named skills are removed.
func UninstallSkillsOpts(ctx context.Context, opts UninstallOptions) error {
	scope := opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}

	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return err
	}

	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine working directory: %w", err)
		}
	}

	state, err := LoadState(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load install state: %w", err)
	}

	if state == nil {
		if scope == ScopeGlobal && hasLegacyInstall(ctx, baseDir) {
			return errors.New("found skills from a previous install without state tracking; run 'databricks experimental aitools install' first, then uninstall")
		}
		return errors.New("no skills installed")
	}

	// Determine which skills to remove.
	var toRemove []string
	if len(opts.Skills) > 0 {
		seen := make(map[string]bool)
		for _, name := range opts.Skills {
			if seen[name] {
				continue
			}
			seen[name] = true
			if _, ok := state.Skills[name]; !ok {
				return fmt.Errorf("skill %q is not installed", name)
			}
			toRemove = append(toRemove, name)
		}
	} else {
		for name := range state.Skills {
			toRemove = append(toRemove, name)
		}
	}

	removeAll := len(toRemove) == len(state.Skills)

	// Remove skill directories and symlinks for each skill.
	for _, name := range toRemove {
		canonicalDir := filepath.Join(baseDir, name)
		removeSymlinksFromAgents(ctx, name, canonicalDir, scope, cwd)
		if err := os.RemoveAll(canonicalDir); err != nil {
			log.Warnf(ctx, "Failed to remove %s: %v", canonicalDir, err)
		}
		delete(state.Skills, name)
	}

	if removeAll {
		// Clean up orphaned symlinks and delete state file.
		cleanOrphanedSymlinks(ctx, baseDir, scope, cwd)
		stateFile := filepath.Join(baseDir, stateFileName)
		if err := os.Remove(stateFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove state file: %w", err)
		}
	} else {
		// Update state to reflect remaining skills.
		if err := SaveState(baseDir, state); err != nil {
			return err
		}
	}

	noun := "skills"
	if len(toRemove) == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Uninstalled %d %s.", len(toRemove), noun))
	return nil
}

// removeSymlinksFromAgents removes a skill's symlink from all agent directories
// in the registry, but only if the entry is a symlink pointing into canonicalDir.
// Non-symlink directories are left untouched to avoid deleting user-managed content.
func removeSymlinksFromAgents(ctx context.Context, skillName, canonicalDir, scope, cwd string) {
	for i := range agents.Registry {
		agent := &agents.Registry[i]
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
			log.Debugf(ctx, "Removed %q from %s", skillName, agent.DisplayName)
		}
	}
}

// cleanOrphanedSymlinks scans all agent skill directories for symlinks pointing
// into baseDir that are not tracked in state, and removes them.
func cleanOrphanedSymlinks(ctx context.Context, baseDir, scope, cwd string) {
	for i := range agents.Registry {
		agent := &agents.Registry[i]
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
