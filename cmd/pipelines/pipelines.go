package pipelines

import (
	"github.com/databricks/bricks/cmd/pipelines/pipelines"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "pipelines",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(pipelines.Cmd)
}
