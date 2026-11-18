package root

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

// Experimental skill-usage tracking: the databricks-core agent skill
// (https://github.com/databricks/databricks-agent-skills/tree/main/skills/databricks-core)
// tells agents to pass the comma-separated bare names of the skills they loaded.
// It may change or be removed without notice.
//
// skillsInvokedFlag is what agents use: a command that still starts with
// "databricks" keeps matching agent permission allowlists such as Claude Code's
// Bash(databricks:*), which an env var prefix does not. It is hidden from help
// and completion so it isn't part of the advertised interface. The env var is
// kept for callers that set it in the environment.
const (
	skillsInvokedFlag   = "skills-invoked"
	skillsInvokedEnvVar = "DATABRICKS_SKILLS_INVOKED"
)

// skillUserAgentKey is repeated once per skill: "skill/databricks-core skill/databricks-jobs".
const skillUserAgentKey = "skill"

const (
	maxSkillLength = 64

	// maxSkillsUserAgentLength bounds all " skill/<name>" pairs together. It
	// fits five of the longest current skill names (45-byte pairs). The usage
	// logs that feed skill analytics cap the user agent at 512 characters, and
	// the rest of the CLI's user agent (auth/, agent/ and other pairs, some
	// appended after these) takes about 200, so the skills must leave room for it.
	maxSkillsUserAgentLength = 225
)

// skillNamePattern is a subset of what the SDK accepts in a user agent value,
// so a valid name can never make useragent.InContext panic.
var skillNamePattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)
})

// parseSkills normalizes a comma-separated list of skill names: entries are
// trimmed, lowercased and deduplicated, invalid ones are dropped, and names are
// kept in order while their pairs fit in maxSkillsUserAgentLength. Invalid input is only logged at debug level because this
// is telemetry and must never fail or warn a command.
func parseSkills(ctx context.Context, raw string) []string {
	var skills []string
	size := 0
	for entry := range strings.SplitSeq(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(entry))
		pairSize := len(" "+skillUserAgentKey+"/") + len(name)
		switch {
		case name == "":
		case len(name) > maxSkillLength || !skillNamePattern().MatchString(name):
			log.Debugf(ctx, "dropping invalid skill entry %q", entry)
		case slices.Contains(skills, name):
		case size+pairSize > maxSkillsUserAgentLength:
			log.Debugf(ctx, "dropping skill entry %q: skills exceed %d user agent bytes", name, maxSkillsUserAgentLength)
		default:
			skills = append(skills, name)
			size += pairSize
		}
	}
	return skills
}

// initSkillsInvokedFlag registers the hidden skillsInvokedFlag on cmd.
func initSkillsInvokedFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String(skillsInvokedFlag, "", "comma-separated agent skills that invoked this command")
	cmd.PersistentFlags().MarkHidden(skillsInvokedFlag)
}

// withSkillsInUserAgent adds one skill/<name> pair per skill from the
// skillsInvokedFlag flag, or skillsInvokedEnvVar when the flag isn't set
// (experimental tracking), unless the user opted out of telemetry.
func withSkillsInUserAgent(ctx context.Context, cmd *cobra.Command) context.Context {
	if telemetry.Disabled(ctx) {
		return ctx
	}
	raw := env.Get(ctx, skillsInvokedEnvVar)
	if f := cmd.Flag(skillsInvokedFlag); f.Changed {
		raw = f.Value.String()
	}
	for _, skill := range parseSkills(ctx, raw) {
		// Each skill adds its own pair to the same context; this is accumulation,
		// not the accidental nesting fatcontext flags.
		ctx = useragent.InContext(ctx, skillUserAgentKey, skill) //nolint:fatcontext
	}
	return ctx
}
