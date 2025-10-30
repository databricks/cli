// Copied to cmd/pipelines/destroy.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package bundle

import (
	"errors"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
	cmd.Flags().BoolVar(&forceDestroy, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return CommandBundleDestroy(cmd, args, autoApprove, forceDestroy)
	}

	return cmd
}

func CommandBundleDestroy(cmd *cobra.Command, args []string, autoApprove, forceDestroy bool) error {
	// we require auto-approve for non tty terminals since interactive consent
	// is not possible
	if !term.IsTerminal(int(os.Stderr.Fd())) && !autoApprove {
		return errors.New("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
	}

	opts := &utils.ProcessOptions{
		InitFunc: func(b *bundle.Bundle) {
			// If `--force-lock` is specified, force acquisition of the deployment lock.
			b.Config.Bundle.Deployment.Lock.Force = forceDestroy

			// If `--auto-approve`` is specified, we skip confirmation checks
			b.AutoApprove = autoApprove
		},
		AlwaysPull: true,
		// Do we need initialize phase here?
	}

	b, err := utils.ProcessBundleWithOut(cmd, opts)
	if err != nil {
		return err
	}

	phases.Destroy(cmd.Context(), b, opts.DirectDeployment)
	if logdiag.HasError(cmd.Context()) {
		return root.ErrAlreadyPrinted
	}

	return nil
}
