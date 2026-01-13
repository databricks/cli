package agent_skills

import "embed"

// SkillsFS embeds all installable agent skills.
//
//go:embed all:databricks-apps
var SkillsFS embed.FS
