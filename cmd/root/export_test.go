package root

import (
	"context"
	"slices"
	"testing"

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

// KnownSkills returns the known skill names.
func KnownSkills() []string { return knownSkills }

// AllowSkillsForTest adds names to the known skills until the test ends, so
// parsing tests can use short or synthetic names.
func AllowSkillsForTest(t *testing.T, names ...string) {
	orig := knownSkills
	knownSkills = append(slices.Clone(orig), names...)
	t.Cleanup(func() { knownSkills = orig })
}

// BuildAll builds every command in t and returns the root command, for tests
// that walk the whole tree.
func BuildAll(ctx context.Context, t *Tree) *cobra.Command {
	return t.buildAll(ctx)
}
