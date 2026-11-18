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

// Experimental skill-usage tracking: agent skills prefix CLI calls with this
// env var so their names reach the user agent. It may change or be removed
// without notice. It is an env var rather than a flag so removing it doesn't
// change the CLI's interface, and because older CLIs ignore unknown env vars
// but reject unknown flags.
const skillsInvokedEnvVar = "DATABRICKS_SKILLS_INVOKED"

// skillUserAgentKey is repeated once per skill: "skill/databricks-core skill/databricks-jobs".
const skillUserAgentKey = "skill"

const (
	maxSkills      = 20
	maxSkillLength = 64
)

// skillNamePattern is a subset of what the SDK accepts in a user agent value,
// so a valid name can never make useragent.InContext panic.
var skillNamePattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)
})

// parseSkills normalizes a comma-separated list of skill names: entries are
// trimmed, lowercased and deduplicated, invalid ones are dropped, and at most
// maxSkills are kept. Invalid input is only logged at debug level because this
// is telemetry and must never fail or warn a command.
func parseSkills(ctx context.Context, raw string) []string {
	var skills []string
	for entry := range strings.SplitSeq(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(entry))
		switch {
		case name == "":
		case len(name) > maxSkillLength || !skillNamePattern().MatchString(name):
			log.Debugf(ctx, "dropping invalid %s entry %q", skillsInvokedEnvVar, entry)
		case slices.Contains(skills, name):
		case len(skills) == maxSkills:
			log.Debugf(ctx, "dropping %s entry %q: more than %d skills", skillsInvokedEnvVar, name, maxSkills)
		default:
			skills = append(skills, name)
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
