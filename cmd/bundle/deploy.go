package bundle

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/spf13/cobra"
)

func newDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy bundle",
		PreRunE: ConfigureBundleWithVariables,
	}

	var forceDeploy bool
	var computeID string
	cmd.Flags().BoolVar(&forceDeploy, "force", false, "Force acquisition of deployment lock.")
	cmd.Flags().StringVarP(&computeID, "compute-id", "c", "", "Override compute in the deployment with the given compute ID.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Force = forceDeploy
		b.Config.Bundle.ComputeID = computeID

		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		))
	}

var forceDeploy bool
var computeID string

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force-override deployment lock and Git branch validation.")
	deployCmd.Flags().StringVarP(&computeID, "compute-id", "c", "", "Override compute in the deployment with the given compute ID.")
}
