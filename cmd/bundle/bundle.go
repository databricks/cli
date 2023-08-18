package bundle

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Databricks Asset Bundles",
	}

	initVariableFlag(cmd)
	cmd.AddCommand(newDeployCommand())
	cmd.AddCommand(newDestroyCommand())
	cmd.AddCommand(newLaunchCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newSchemaCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newTestCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newInitCommand())
	return cmd
}
