package installer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/log"
)

// RemoveLegacyRawSkills removes raw skill directories the CLI previously placed
// in a plugin agent's skills dir, so installing the plugin doesn't leave the
// same skills duplicated as loose files. It only removes our own content: a
// symlink into our canonical dir, or a copied skill whose contents exactly match
// the recorded checksums. User-modified or third-party directories, dirs with
// extra files, and copies with no recorded provenance are left untouched.
func RemoveLegacyRawSkills(ctx context.Context, agent *agents.Agent, scope string) error {
	entries, err := managedRawSkillEntriesForAgent(ctx, agent, scope)
	if err != nil {
		return err
	}

	for _, entryPath := range entries {
		// RemoveAll on a symlink removes the link, not its target.
		if err := os.RemoveAll(entryPath); err != nil {
			log.Warnf(ctx, "Failed to remove legacy skill %s: %v", entryPath, err)
		} else {
			log.Debugf(ctx, "Removed legacy skill %s for %s", filepath.Base(entryPath), agent.DisplayName)
		}
	}
	return nil
}

// HasManagedRawSkillsForAgent reports whether an agent has raw skill exposure
// that this CLI can safely remove after migrating that agent to the plugin.
func HasManagedRawSkillsForAgent(ctx context.Context, agent *agents.Agent, scope string) (bool, error) {
	entries, err := managedRawSkillEntriesForAgent(ctx, agent, scope)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func managedRawSkillEntriesForAgent(ctx context.Context, agent *agents.Agent, scope string) ([]string, error) {
	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return nil, err
	}
	state, err := LoadState(baseDir)
	if err != nil {
		return nil, err
	}

	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(agentDir)
	if errors.Is(err, fs.ErrNotExist) {
		// No skills dir for this agent; nothing to clean up.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var managed []string
	for _, entry := range entries {
		entryPath := filepath.Join(agentDir, entry.Name())
		canonicalDir := filepath.Join(baseDir, entry.Name())
		if !exposureRemovable(entryPath, canonicalDir, entry.Name(), state) {
			continue
		}
		managed = append(managed, entryPath)
	}
	return managed, nil
}
