package bundle

import (
	"fmt"
	"os"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy deployed bundle resources",

	PreRunE: root.MustConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = force

		// If `--auto-approve`` is specified, we skip confirmation checks
		b.AutoApprove = autoApprove

		// we require auto-approve for non tty terminals since interactive consent
		// is not possible
		if !term.IsTerminal(int(os.Stderr.Fd())) && !autoApprove {
			return fmt.Errorf("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
		}

		// TODO: remove once state for inplace is moved to event
		logger, ok := cmdio.FromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("progress logger not found")
		}
		if logger.Mode == flags.ModeInplace {
			logger.Mode = flags.ModeAppend
		}

		return bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Destroy(),
		})
	},
}

var autoApprove bool

func init() {
	AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
}
