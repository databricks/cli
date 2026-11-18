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
)

// Experimental skill-usage tracking: the databricks-core agent skill
// (https://github.com/databricks/databricks-agent-skills/tree/main/skills/databricks-core)
// tells agents to prefix CLI calls with this env var, set to the comma-separated
// bare names of the skills they loaded. It may change or be removed without
// notice. It is an env var rather than a flag so removing it doesn't change the
// CLI's interface, and because older CLIs ignore unknown env vars but reject
// unknown flags.
const skillsInvokedEnvVar = "DATABRICKS_SKILLS_INVOKED"

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
			log.Debugf(ctx, "dropping invalid %s entry %q", skillsInvokedEnvVar, entry)
		case slices.Contains(skills, name):
		case size+pairSize > maxSkillsUserAgentLength:
			log.Debugf(ctx, "dropping %s entry %q: skills exceed %d user agent bytes", skillsInvokedEnvVar, name, maxSkillsUserAgentLength)
		default:
			skills = append(skills, name)
			size += pairSize
		}
	}
	return skills
}

// withSkillsInUserAgent adds one skill/<name> pair per skill in
// skillsInvokedEnvVar (experimental tracking), unless the user opted out of
// telemetry.
func withSkillsInUserAgent(ctx context.Context) context.Context {
	if telemetry.Disabled(ctx) {
		return ctx
	}
	for _, skill := range parseSkills(ctx, env.Get(ctx, skillsInvokedEnvVar)) {
		// Each skill adds its own pair to the same context; this is accumulation,
		// not the accidental nesting fatcontext flags.
		ctx = useragent.InContext(ctx, skillUserAgentKey, skill) //nolint:fatcontext
	}
	return ctx
}
