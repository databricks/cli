package aitools

import (
	"context"
	"strings"

	"github.com/databricks/cli/cmd/root"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/spf13/cobra"
)

// Spec declares the aitools command.
var Spec = root.CommandSpec{
	Use:   "aitools",
	Short: "Databricks skills and plugins for coding agents",
	Build: buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Install Databricks skills and plugins into your coding agent so it can work
effectively with Databricks resources (bundles, jobs, SQL, and more).

Supported agents: ` + strings.Join(agents.SupportedNames(), ", ") + `.

Skills and plugins are sourced from
https://github.com/databricks/databricks-agent-skills`

	cmd.AddCommand(NewInstallCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewUninstallCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewVersionCmd())
}
