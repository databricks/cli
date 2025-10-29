package deployment

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind KEY RESOURCE_ID",
		Short: "Bind bundle-defined resources to existing resources",
		Long: `Bind a resource in your bundle to an existing resource in the workspace.

This command binds a workspace resource to a corresponding bundle resource definition.
After binding, the workspace resource will be updated based on the bundle configuration on the next deployment.

Arguments:
  KEY         - The resource key defined in your bundle configuration
  RESOURCE_ID - The ID of the existing resource in the workspace

Examples:
  # Bind a job resource to existing workspace job
  databricks bundle deployment bind my_etl_job 6565621249

  # Bind a pipeline to existing Lakeflow Declarative Pipeline
  databricks bundle deployment bind data_pipeline 9876543210

  # Bind with automatic approval (useful for CI/CD)
  databricks bundle deployment bind my_job 123 --auto-approve

Common workflow:
1. First, generate bundle configuration from an existing resource:
   databricks bundle generate job --existing-job-id 6565621249 --key my_etl_job

2. Then bind the bundle resource to the workspace resource:
   databricks bundle deployment bind my_etl_job 6565621249

3. Deploy to apply bundle configuration to the bound resource:
   databricks bundle deploy

WARNING: After binding, the workspace resource will be managed by your bundle.
Any manual changes made in the workspace UI may be overwritten on deployment.`,
		Args: root.ExactArgs(2),
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := BindResource(cmd, args[0], args[1], autoApprove, forceLock, false)
		if err != nil {
			return err
		}

		cmdio.LogString(cmd.Context(), "Run 'bundle deploy' to deploy changes to your workspace")
		return nil
	}

	return cmd
}
