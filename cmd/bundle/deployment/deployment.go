package deployment

import (
	"github.com/spf13/cobra"
)

func NewDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "Deployment related commands",
		Long: `Deployment related commands for managing bundle resource bindings.

Use these commands to link / unlink bundle definitions to existing workspace resources.

COMMON WORKFLOW:
1. Generate configuration from existing resource:
   databricks bundle generate job --existing-job-id 12345 --key my_job

2. Bind the bundle resource to the existing workspace resource:
   databricks bundle deployment bind my_job 12345

3. Deploy updates - the bound resource will be updated in the workspace:
   databricks bundle deploy

After binding, the existing workspace resource will be managed by your bundle.`,
	}

	cmd.AddCommand(newBindCommand())
	cmd.AddCommand(newUnbindCommand())
	return cmd
}
