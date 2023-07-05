package bundle

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: ConfigureBundleWithVariables,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = forceDeploy

		// If `--auto-approve`` is specified, we skip confirmation checks
		b.AutoApprove = autoApproveDeploy

		// we require auto-approve for non tty terminals since interactive consent
		// is not possible
		if !cmdio.IsErrTTY(cmd.Context()) && !autoApproveDeploy {
			return fmt.Errorf("please specify --auto-approve to skip interactive confirmation checks for non tty consoles")
		}

		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		))
	},
}

var forceDeploy bool
var autoApproveDeploy bool

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&autoApproveDeploy, "auto-approve", false, "Skip interactive approvals for deployment")
	deployCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force acquisition of deployment lock.")
}
