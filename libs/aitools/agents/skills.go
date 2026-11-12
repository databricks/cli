package agents

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
)

const (
	// databricksSkillPrefix is the prefix used by Databricks skills (e.g., "databricks", "databricks-apps").
	databricksSkillPrefix = "databricks"

	// CanonicalSkillsDir is the shared location for skills when multiple agents are detected.
	CanonicalSkillsDir = ".databricks/aitools/skills"

	// LegacySkillsDir is the old canonical location, checked for backward compatibility.
	LegacySkillsDir = ".databricks/agent-skills"
)

// HasDatabricksToolingInstalled reports whether every detected agent already
// has Databricks tooling, so `apps init` has nothing to recommend. Skills in
// the shared canonical (or legacy) dir cover every agent. Otherwise each
// detected agent needs the databricks plugin in its own plugin manifest,
// installed for every project or for the working directory (see
// pluginInstall.appliesTo): a plugin installed for another project does not
// load for the app being created. It also reports true when no agent is
// detected.
func HasDatabricksToolingInstalled(ctx context.Context) bool {
	installed := DetectInstalled(ctx)
	if len(installed) == 0 || hasSharedSkills(ctx) {
		return true
	}
	cwd := ""
	if wd, err := os.Getwd(); err == nil {
		cwd = resolvePath(wd)
	}
	for _, a := range installed {
		if a.pluginInstalls == nil || !a.databricksPluginActiveIn(ctx, cwd) {
			return false
		}
	}
	return true
}

// hasSharedSkills reports whether Databricks skills are in the canonical
// skills dir or the legacy one.
func hasSharedSkills(ctx context.Context) bool {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	return HasDatabricksSkillsIn(filepath.Join(homeDir, CanonicalSkillsDir)) ||
		HasDatabricksSkillsIn(filepath.Join(homeDir, LegacySkillsDir))
}

// HasDatabricksSkillsIn checks if dir contains a subdirectory starting with "databricks".
// The CLI installs skills into an agent's skills dir as symlinks to the canonical
// store, and os.ReadDir reports symlinks via Lstat (so IsDir is false for them), so
// entries are resolved with os.Stat to follow the link to its target.
func HasDatabricksSkillsIn(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), databricksSkillPrefix) {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, e.Name()))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
