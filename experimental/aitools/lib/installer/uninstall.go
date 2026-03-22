package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// UninstallSkills removes all installed skills, their symlinks, and the state file.
func UninstallSkills(ctx context.Context) error {
	globalDir, err := GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}

	state, err := LoadState(globalDir)
	if err != nil {
		return fmt.Errorf("failed to load install state: %w", err)
	}

	if state == nil {
		if hasLegacyInstall(ctx, globalDir) {
			return fmt.Errorf("found skills from a previous install without state tracking; run 'databricks experimental aitools install' first, then uninstall")
		}
		return fmt.Errorf("no skills installed")
	}

	skillCount := len(state.Skills)

	// Remove skill directories and symlinks for each skill in state.
	for name := range state.Skills {
		canonicalDir := filepath.Join(globalDir, name)

		// Remove symlinks from agent directories (only symlinks pointing to canonical dir).
		removeSymlinksFromAgents(ctx, name, canonicalDir)

		// Remove canonical skill directory.
		if err := os.RemoveAll(canonicalDir); err != nil {
			log.Warnf(ctx, "Failed to remove %s: %v", canonicalDir, err)
		}
	}

	// Clean up orphaned symlinks pointing into the canonical dir.
	cleanOrphanedSymlinks(ctx, globalDir)

	// Delete state file.
	stateFile := filepath.Join(globalDir, stateFileName)
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	noun := "skills"
	if skillCount == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Uninstalled %d %s.", skillCount, noun))
	return nil
}

// removeSymlinksFromAgents removes a skill's symlink from all agent directories
// in the registry, but only if the entry is a symlink pointing into canonicalDir.
// Non-symlink directories are left untouched to avoid deleting user-managed content.
func removeSymlinksFromAgents(ctx context.Context, skillName, canonicalDir string) {
	for i := range agents.Registry {
		agent := &agents.Registry[i]
		skillsDir, err := agent.SkillsDir(ctx)
		if err != nil {
			continue
		}

		destDir := filepath.Join(skillsDir, skillName)

		// Use Lstat to detect symlinks (Stat follows them).
		fi, err := os.Lstat(destDir)
		if os.IsNotExist(err) {
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

		// Only remove if the symlink points into our canonical dir.
		if !strings.HasPrefix(target, canonicalDir+string(os.PathSeparator)) && target != canonicalDir {
			log.Debugf(ctx, "Skipping symlink %s (points to %s, not %s)", destDir, target, canonicalDir)
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
// into globalDir that are not tracked in state, and removes them.
func cleanOrphanedSymlinks(ctx context.Context, globalDir string) {
	for i := range agents.Registry {
		agent := &agents.Registry[i]
		skillsDir, err := agent.SkillsDir(ctx)
		if err != nil {
			continue
		}

		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			entryPath := filepath.Join(skillsDir, entry.Name())

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

			// Check if the symlink points into our global skills dir.
			if !strings.HasPrefix(target, globalDir+string(os.PathSeparator)) && target != globalDir {
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
