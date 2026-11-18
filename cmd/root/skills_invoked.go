package root

import (
	"context"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/spf13/cobra"
)

// Agent skills set envSkillsInvoked rather than the flag because older CLIs
// reject unknown flags but ignore unknown environment variables.
const envSkillsInvoked = "DATABRICKS_SKILLS_INVOKED"

const flagSkillsInvoked = "skills-invoked"

func initSkillsInvokedFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagSkillsInvoked, "", "comma-separated agent skills that invoked this command")
	cmd.PersistentFlags().MarkHidden(flagSkillsInvoked)
}

// withSkillsInvoked stores the skills that drove cmd in ctx. The flag wins over
// the environment variable.
func withSkillsInvoked(ctx context.Context, cmd *cobra.Command) context.Context {
	raw := env.Get(ctx, envSkillsInvoked)
	if v := cmd.Flag(flagSkillsInvoked).Value.String(); v != "" {
		raw = v
	}
	return skillsinvoked.WithSkills(ctx, skillsinvoked.Parse(ctx, raw))
}
