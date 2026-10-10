// Copied to cmd/pipelines/destroy.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package bundle

import (
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newDestroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy deployed bundle resources",
		Long: `Destroy all resources deployed by this bundle from the workspace.

This command removes all Databricks resources that were created by deploying
this bundle.

Examples:
  databricks bundle destroy                 # Destroy resources in default target
  databricks bundle destroy --target prod   # Destroy resources in production target

Typical use cases:
- Cleaning up development or testing targets
- Removing resources during environment decommissioning`,
		Args: root.NoArgs,
	}

	var autoApprove bool
	var forceDestroy bool
	var quiet int
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
	cmd.Flags().BoolVar(&forceDestroy, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().CountVarP(&quiet, "quiet", "q", "Reduce output: -qq prints only warnings and errors.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return utils.CommandBundleDestroy(cmd, args, autoApprove, forceDestroy)
	}

	return cmd
}
