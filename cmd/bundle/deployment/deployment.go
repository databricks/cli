package deployment

import (
	"github.com/databricks/cli/clis"
	"github.com/spf13/cobra"
)

func NewDeploymentCommand(hidden bool, cliType clis.CLIType) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "deployment",
		Short:  "Deployment related commands",
		Long:   "Deployment related commands",
		Hidden: hidden,
	}

	cmd.AddCommand(newBindCommand(cliType))
	cmd.AddCommand(newUnbindCommand(cliType))
	return cmd
}
