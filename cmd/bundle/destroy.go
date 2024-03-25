package bundle

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newDestroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy deployed bundle resources",
		Args:    root.NoArgs,
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	var autoApprove bool
	var forceDestroy bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
	cmd.Flags().BoolVar(&forceDestroy, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)

		bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
			// If `--force-lock` is specified, force acquisition of the deployment lock.
			b.Config.Bundle.Deployment.Lock.Force = forceDestroy

			// If `--auto-approve`` is specified, we skip confirmation checks
			b.AutoApprove = autoApprove

			return nil
		})

		// we require auto-approve for non tty terminals since interactive consent
		// is not possible
		if !term.IsTerminal(int(os.Stderr.Fd())) && !autoApprove {
			return fmt.Errorf("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
		}

		// Check auto-approve is selected for json logging
		logger, ok := cmdio.FromContext(ctx)
		if !ok {
			return fmt.Errorf("progress logger not found")
		}
		if logger.Mode == flags.ModeJson && !autoApprove {
			return fmt.Errorf("please specify --auto-approve since selected logging format is json")
		}

		diags := bundle.Apply(ctx, b, bundle.Seq(
			phases.Initialize(),
			phases.Build(),
			phases.Destroy(),
		))
		if err := diags.Error(); err != nil {
			return err
		}
		return nil
	}

	return cmd
}
