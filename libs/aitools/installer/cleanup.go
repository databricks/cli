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
		canonicalDir := filepath.Join(baseDir, entry.Name())
		if !exposureRemovable(entryPath, canonicalDir, entry.Name(), state) {
			continue
		}
		// RemoveAll on a symlink removes the link, not its target.
		if err := os.RemoveAll(entryPath); err != nil {
			log.Warnf(ctx, "Failed to remove legacy skill %s: %v", entryPath, err)
		} else {
			log.Debugf(ctx, "Removed legacy skill %s for %s", entry.Name(), agent.DisplayName)
		}
	}
	return nil
}
