package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: root.MustConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = force

		// deployment does not support inplace logging
		ctx = cmdio.DisableInplace(ctx)

		return bundle.Apply(ctx, b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		})
	},
}

var force bool

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&force, "force", false, "Force acquisition of deployment lock.")
}
