package bundle

import (
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

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = forceDeploy

		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		))
	},
}

var forceDeploy bool

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force acquisition of deployment lock.")
}
