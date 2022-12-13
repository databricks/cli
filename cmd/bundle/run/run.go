package run

import (
	"github.com/spf13/cobra"

	parent "github.com/databricks/bricks/cmd/bundle"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a workload (e.g. a job or a pipeline)",
}

func AddCommand(cmd *cobra.Command) {
	runCmd.AddCommand(cmd)
}

func init() {
	parent.AddCommand(runCmd)
}
