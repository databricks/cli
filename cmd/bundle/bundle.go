package bundle

import (
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Databricks Asset Bundles let you express data/AI/analytics projects as code.",
		Long: `Databricks Asset Bundles let you express data/AI/analytics projects as code.

Common workflows:
  databricks bundle init default-python      # Initialize new project
  databricks bundle deploy --target dev      # Deploy to development
  databricks bundle run my_job               # Run jobs/pipelines
  databricks bundle deploy --target prod     # Deploy to production

Import existing resources:
  databricks bundle generate job --existing-job-id 123 --key my_job # Generate job configuration
  databricks bundle deployment bind my_job 123                      # Link to an existing job

Online documentation: https://docs.databricks.com/en/dev-tools/bundles/index.html`,
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
	cmd.AddCommand(newPlanCommand())
	return cmd
}
