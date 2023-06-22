package bundle

import (
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: ConfigureBundleWithVariables,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		return deploy(cmd, b)
	},
}

func deploy(cmd *cobra.Command, b *bundle.Bundle) error {
	// If `--force` is specified, force acquisition of the deployment lock.
	b.Config.Bundle.Lock.Force = forceDeploy

	if computeID == "" {
		computeID = os.Getenv("DATABRICKS_CLUSTER_ID")
	}

	return bundle.Apply(cmd.Context(), b, bundle.Seq(
		phases.Initialize(computeID),
		phases.Build(),
		phases.Deploy(),
	))
}

var forceDeploy bool
var computeID string

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force acquisition of deployment lock.")
	deployCmd.Flags().StringVar(&computeID, "compute", "", "Override compute in the deployment with the given compute ID.")
}
