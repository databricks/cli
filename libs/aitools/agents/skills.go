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

	// legacySkillsDir is the old canonical location, checked for backward compatibility.
	legacySkillsDir = ".databricks/agent-skills"
)

// HasDatabricksSkillsInstalled checks if Databricks skills are installed in the canonical location.
// Returns true if no agents are detected (nothing to recommend) or if skills exist in
// ~/.databricks/aitools/skills/ or the legacy ~/.databricks/agent-skills/.
func HasDatabricksSkillsInstalled(ctx context.Context) bool {
	installed := DetectInstalled(ctx)
	if len(installed) == 0 {
		return true
	}

	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	return hasDatabricksSkillsIn(filepath.Join(homeDir, CanonicalSkillsDir)) ||
		hasDatabricksSkillsIn(filepath.Join(homeDir, legacySkillsDir))
}

// hasDatabricksSkillsIn checks if dir contains a subdirectory starting with "databricks".
func hasDatabricksSkillsIn(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), databricksSkillPrefix) {
			return true
		}
	}
	return false
}
