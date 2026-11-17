package root

import (
	"context"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/spf13/cobra"
)

// envSkillsInvoked is the primary channel: skills prefix commands with it, and
// older CLIs silently ignore unknown environment variables.
const envSkillsInvoked = "DATABRICKS_SKILLS_INVOKED"

const flagSkillsInvoked = "skills-invoked"

func initSkillsInvokedFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagSkillsInvoked, "", "comma-separated agent skills that invoked this command")
	cmd.PersistentFlags().MarkHidden(flagSkillsInvoked)
}

// withSkillsInvoked stores the skills that drove cmd in ctx. The flag wins over
// the environment variable when both are set.
func withSkillsInvoked(ctx context.Context, cmd *cobra.Command) context.Context {
	raw := env.Get(ctx, envSkillsInvoked)
	if f := cmd.Flag(flagSkillsInvoked); f != nil && f.Value.String() != "" {
		raw = f.Value.String()
	}
	return skillsinvoked.WithSkills(ctx, skillsinvoked.Parse(ctx, raw))
}
