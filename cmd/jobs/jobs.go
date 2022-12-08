package jobs

import (
	"github.com/databricks/bricks/cmd/jobs/jobs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "jobs",
	Short: `The Jobs API allows you to create, edit, and delete jobs.`,
	Long:  `The Jobs API allows you to create, edit, and delete jobs.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(jobs.Cmd)
}
