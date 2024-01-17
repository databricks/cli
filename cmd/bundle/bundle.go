package bundle

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Databricks Asset Bundles let you express data/AI/analytics projects as code.",
		Long:    "Databricks Asset Bundles let you express data/AI/analytics projects as code.\n\nOnline documentation: https://docs.databricks.com/en/dev-tools/bundles",
		GroupID: "development",
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
	cmd.AddCommand(newSummariseCommand())
	cmd.AddCommand(newGenerateCommand())
	return cmd
}
