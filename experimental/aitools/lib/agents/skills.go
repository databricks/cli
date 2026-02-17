package agents

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// databricksSkillPrefix is the prefix used by Databricks skills (e.g., "databricks", "databricks-apps").
	databricksSkillPrefix = "databricks"

	// canonicalSkillsDir is the shared location for skills when multiple agents are detected.
	canonicalSkillsDir = ".databricks/agent-skills"
)

// HasDatabricksSkillsInstalled checks if at least one detected agent has Databricks skills installed.
// Returns true if no agents are detected (nothing to recommend) or if any agent has Databricks skills.
func HasDatabricksSkillsInstalled() bool {
	installed := DetectInstalled()
	if len(installed) == 0 {
		return true
	}

	// Check canonical location first (~/.databricks/agent-skills/).
	homeDir, err := getHomeDir()
	if err == nil {
		if hasDatabricksSkillsIn(filepath.Join(homeDir, canonicalSkillsDir)) {
			return true
		}
	}

	// Check each agent's skills directory.
	for _, agent := range installed {
		dir, err := agent.SkillsDir()
		if err != nil {
			continue
		}
		if hasDatabricksSkillsIn(dir) {
			return true
		}
	}
	return false
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
