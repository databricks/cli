package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/progress"
	"github.com/spf13/cobra"
)

// TODO: do we need ci/cd for non-tty

// TODO:
// 1. Delete resources
// 2. Delete files - need tracking dirs for this
// 3. Delete artifacts
// 4. What about running resources

// TODO: json logs?

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy deployed bundle resources",

	PreRunE: root.MustConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// If `--force` is specified, force acquisition of the deployment lock.
		b.Config.Bundle.Lock.Force = force

		ctx := progress.NewContext(cmd.Context(), progress.NewLogger(flags.ModeAppend))
		return bundle.Apply(ctx, b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Destroy(),
		})
	},
}

var skipConfirmation bool

func init() {
	AddCommand(destroyCmd)
	deployCmd.Flags().BoolVar(&skipConfirmation, "skip-confirmation", false, "skip confirmation before destroy")
}
