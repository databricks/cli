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

		return Deploy(cmd, b)
	},
}

func Deploy(cmd *cobra.Command, b *bundle.Bundle) error {
	// If `--force` is specified, force acquisition of the deployment lock.
	b.Config.Bundle.Lock.Force = force

	if computeID == "" {
		computeID = os.Getenv("DATABRICKS_COMPUTE")
	}

	return bundle.Apply(cmd.Context(), b, bundle.Seq(
		phases.Initialize(computeID),
		phases.Build(),
		phases.Deploy(),
	))
}

var force bool
var computeID string

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&force, "force", false, "Force acquisition of deployment lock.")
	deployCmd.Flags().StringVar(&computeID, "compute", "", "Override compute in the deployment with the given compute ID.")
}
