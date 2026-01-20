package init_template

import (
	"github.com/spf13/cobra"
)

// NewInitTemplateCommand creates a command group for initializing project templates.
func NewInitTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-template",
		Short: "Initialize project templates",
		Long: `Initialize project templates for Databricks resources.

Subcommands:
  app       Initialize a Databricks App using the appkit template
  job       Initialize a job project using the default-python template
  pipeline  Initialize a Lakeflow pipeline project
  empty     Initialize an empty bundle for custom resources (dashboards, alerts, etc.)`,
	}
	cmd.AddCommand(newAppCmd())
	cmd.AddCommand(newJobCmd())
	cmd.AddCommand(newPipelineCmd())
	cmd.AddCommand(newEmptyCmd())
	return cmd
}
