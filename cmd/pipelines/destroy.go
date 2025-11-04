// Copied from cmd/bundle/destroy.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"github.com/databricks/cli/cmd/bundle"
	"github.com/spf13/cobra"
)

func destroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a pipelines project",
		Args:  cobra.NoArgs,
	}

	var autoApprove bool
	var forceDestroy bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting pipelines.")
	cmd.Flags().BoolVar(&forceDestroy, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return bundle.CommandBundleDestroy(cmd, args, autoApprove, forceDestroy)
	}

	return cmd
}
