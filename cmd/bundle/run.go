package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/deploy/terraform"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/bundle/run"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags] KEY",
	Short: "Run a workload (e.g. a job or a pipeline)",

	Args:    cobra.ExactArgs(1),
	PreRunE: ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		err := bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			terraform.Initialize(),
			terraform.Load(),
		})
		if err != nil {
			return err
		}

		runner, err := run.Find(b, args[0])
		if err != nil {
			return err
		}

		err = runner.Run(cmd.Context())
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
