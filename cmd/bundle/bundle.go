package bundle

import (
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Databricks Asset Bundles let you express data/AI/analytics projects as code.",
		Long:    "Databricks Asset Bundles let you express data/AI/analytics projects as code.\n\nOnline documentation: https://docs.databricks.com/en/dev-tools/bundles/index.html",
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
	cmd.AddCommand(newSummaryCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newDebugCommand())
	cmd.AddCommand(deployment.NewDeploymentCommand())
	cmd.AddCommand(newOpenCommand())
	return cmd
}
