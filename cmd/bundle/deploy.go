package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	PreRunE: ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		return bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			phases.Build(),
			phases.Deploy(),
		})
	},
}

func init() {
	AddCommand(deployCmd)
}
