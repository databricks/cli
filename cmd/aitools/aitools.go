package aitools

import (
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aitools",
		Short: "Databricks skills and plugins for coding agents",
		Long: `Install Databricks skills and plugins into your coding agent so it can work
effectively with Databricks resources (bundles, jobs, SQL, and more).

Supported agents: ` + strings.Join(agents.SupportedNames(), ", ") + `.

Skills and plugins are sourced from
https://github.com/databricks/databricks-agent-skills`,
	}

	cmd.AddCommand(NewInstallCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewUninstallCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewVersionCmd())

	return cmd
}
