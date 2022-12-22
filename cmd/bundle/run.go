package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/deploy/terraform"
	"github.com/databricks/bricks/bundle/phases"
	"github.com/databricks/bricks/bundle/run"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags] KEY...",
	Short: "Run a workload (e.g. a job or a pipeline)",

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

		runners, err := run.Collect(b, args)
		if err != nil {
			return err
		}

		for _, runner := range runners {
			err = runner.Run(cmd.Context())
			if err != nil {
				return err
			}
		}

		return nil
	},

	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		b, err := LoadAndSelectEnvironment(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return run.ResourceCompletions(b), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
