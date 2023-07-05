package bundle

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy deployed bundle resources",

	PreRunE: ConfigureBundleWithVariables,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = forceDestroy

		// If `--auto-approve`` is specified, we skip confirmation checks
		b.AutoApprove = autoApproveDestroy

		// we require auto-approve for non tty terminals since interactive consent
		// is not possible
		if !cmdio.IsErrTTY(cmd.Context()) && !autoApproveDestroy {
			return fmt.Errorf("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
		}

		// Check auto-approve is selected for json logging
		logger, ok := cmdio.FromContext(ctx)
		if !ok {
			return fmt.Errorf("progress logger not found")
		}
		if logger.Mode == flags.ModeJson && !autoApproveDestroy {
			return fmt.Errorf("please specify --auto-approve since selected logging format is json")
		}

		return bundle.Apply(ctx, b, bundle.Seq(
			phases.Initialize(),
			phases.Build(),
			phases.Destroy(),
		))
	},
}

var autoApproveDestroy bool
var forceDestroy bool

func init() {
	AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&autoApproveDestroy, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
	destroyCmd.Flags().BoolVar(&forceDestroy, "force", false, "Force acquisition of deployment lock.")
}
