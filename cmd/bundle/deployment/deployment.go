package deployment

import (
	"github.com/spf13/cobra"
)

func NewDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "Deployment related commands",
		Long:  "Deployment related commands",
	}

	cmd.AddCommand(newBindCommand())
	cmd.AddCommand(newUnbindCommand())
	return cmd
}
