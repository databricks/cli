package root

import (
	"context"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/databricks/cli/libs/telemetry"
)

// envSkillsInvoked is an env var rather than a flag so it can be removed
// without breaking the CLI's interface, and because older CLIs ignore unknown
// env vars but reject unknown flags.
const envSkillsInvoked = "DATABRICKS_SKILLS_INVOKED"

// withSkillsInvoked stores the skills from envSkillsInvoked in ctx, unless the
// user opted out of telemetry.
func withSkillsInvoked(ctx context.Context) context.Context {
	if telemetry.Disabled(ctx) {
		return ctx
	}
	return skillsinvoked.WithSkills(ctx, skillsinvoked.Parse(ctx, env.Get(ctx, envSkillsInvoked)))
}
