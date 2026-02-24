package agents

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// databricksSkillPrefix is the prefix used by Databricks skills (e.g., "databricks", "databricks-apps").
	databricksSkillPrefix = "databricks"

	// CanonicalSkillsDir is the shared location for skills when multiple agents are detected.
	CanonicalSkillsDir = ".databricks/agent-skills"
)

// HasDatabricksSkillsInstalled checks if Databricks skills are installed in the canonical location.
// Returns true if no agents are detected (nothing to recommend) or if skills exist in ~/.databricks/agent-skills/.
// Only the canonical location is checked so that skills installed by other tools are not mistaken for a proper installation.
func HasDatabricksSkillsInstalled() bool {
	installed := DetectInstalled()
	if len(installed) == 0 {
		return true
	}

	homeDir, err := getHomeDir()
	if err != nil {
		return false
	}
	return hasDatabricksSkillsIn(filepath.Join(homeDir, CanonicalSkillsDir))
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
