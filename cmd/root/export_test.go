package root

import (
	"context"

	"github.com/spf13/cobra"
)

// Experimental skill-usage tracking, see user_agent_skills.go.
var (
	ParseSkills              = parseSkills
	WithSkillsInUserAgent    = withSkillsInUserAgent
	SkillsInvokedEnvVar      = skillsInvokedEnvVar
	InitSkillsInvokedFlag    = initSkillsInvokedFlag
	MaxSkillsUserAgentLength = maxSkillsUserAgentLength
)

// BuildAll builds every command in t and returns the root command, for tests
// that walk the whole tree.
func BuildAll(ctx context.Context, t *Tree) *cobra.Command {
	return t.buildAll(ctx)
}
