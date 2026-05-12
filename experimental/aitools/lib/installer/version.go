package installer

import (
	_ "embed"
	"strings"
)

//go:embed SKILLS_VERSION
var skillsVersionFile string

// defaultSkillsRepoRef is the pinned tag of databricks/databricks-agent-skills.
// It is sourced from the SKILLS_VERSION file so automation can bump the pin
// with a single-line file edit instead of patching Go source.
var defaultSkillsRepoRef = strings.TrimSpace(skillsVersionFile)
