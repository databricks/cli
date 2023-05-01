package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/flags"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: root.MustConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// Initialize variables from command line values
		b.Config.InitializeVariables(deployVariables)

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = force

		return bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		})
	},
}

var force bool

var deployVariables []string

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&force, "force", false, "Force acquisition of deployment lock.")
	flags.AddVariableFlag(deployCmd, &deployVariables)
}
