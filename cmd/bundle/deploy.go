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

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: root.MustConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(cmd.Context())

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = forceDeploy

		// If `--auto-approve`` is specified, we skip confirmation checks
		b.AutoApprove = autoApproveDeploy

		// we require auto-approve for non tty terminals since interactive consent
		// is not possible
		if !term.IsTerminal(int(os.Stderr.Fd())) && !autoApproveDeploy {
			return fmt.Errorf("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
		}

		// Check auto-approve is selected for json logging
		logger, ok := cmdio.FromContext(ctx)
		if !ok {
			return fmt.Errorf("progress logger not found")
		}
		if logger.Mode == flags.ModeJson && !autoApproveDeploy {
			return fmt.Errorf("please specify --auto-approve since selected logging format is json")
		}

		return bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		})
	},
}

var forceDeploy bool

var autoApproveDeploy bool

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force acquisition of deployment lock.")
	deployCmd.Flags().BoolVar(&autoApproveDeploy, "auto-approve", false, "Skip interactive approvals")
}
