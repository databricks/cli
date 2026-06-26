package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/log"
)

// RemoveLegacyRawSkills removes raw skill directories the CLI previously placed
// in a plugin agent's skills dir, so installing the plugin doesn't leave the
// same skills duplicated as loose files. It only removes our own content: a
// symlink pointing into our canonical dir, or a copied skill whose files all
// match the checksums we recorded. User-modified or third-party directories,
// and copies with no recorded provenance, are left untouched.
func RemoveLegacyRawSkills(ctx context.Context, agent *agents.Agent, scope string) error {
	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return err
	}
	state, err := LoadState(baseDir)
	if err != nil {
		return err
	}

	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(agentDir)
	if errors.Is(err, fs.ErrNotExist) {
		// No skills dir for this agent; nothing to clean up.
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(agentDir, entry.Name())
		fi, err := os.Lstat(entryPath)
		if err != nil {
			continue
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			if symlinkPointsInto(entryPath, baseDir) {
				if err := os.Remove(entryPath); err != nil {
					log.Warnf(ctx, "Failed to remove legacy skill symlink %s: %v", entryPath, err)
				} else {
					log.Debugf(ctx, "Removed legacy skill symlink %s for %s", entry.Name(), agent.DisplayName)
				}
			}
			continue
		}

		if fi.IsDir() && state != nil && skillCopyUnmodified(entryPath, entry.Name(), state) {
			if err := os.RemoveAll(entryPath); err != nil {
				log.Warnf(ctx, "Failed to remove legacy skill %s: %v", entryPath, err)
			} else {
				log.Debugf(ctx, "Removed legacy skill copy %s for %s", entry.Name(), agent.DisplayName)
			}
		}
	}
	return nil
}

// symlinkPointsInto reports whether linkPath is a symlink resolving into baseDir.
func symlinkPointsInto(linkPath, baseDir string) bool {
	target, err := os.Readlink(linkPath)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Clean(filepath.Join(filepath.Dir(linkPath), target))
	}
	return target == baseDir || strings.HasPrefix(target, baseDir+string(os.PathSeparator))
}

// skillCopyUnmodified reports whether every file the CLI recorded for a skill is
// present in dir with its recorded sha256. Returns false when there is no
// recorded provenance, so unverifiable copies are kept.
func skillCopyUnmodified(dir, skillName string, state *InstallState) bool {
	prefix := skillName + "/"
	var recorded []string
	for path := range state.Files {
		if strings.HasPrefix(path, prefix) {
			recorded = append(recorded, path)
		}
	}
	if len(recorded) == 0 {
		return false
	}
	for _, relPath := range recorded {
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(relPath, prefix)))
		data, err := os.ReadFile(file)
		if err != nil {
			return false
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != state.Files[relPath].SHA256 {
			return false
		}
	}
	return true
}
