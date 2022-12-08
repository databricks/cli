package jobs

import (
	"github.com/databricks/bricks/cmd/jobs/jobs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "jobs",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(jobs.Cmd)
}
