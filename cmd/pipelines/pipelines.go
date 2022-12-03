package pipelines

import (
	"github.com/databricks/bricks/cmd/pipelines/pipelines"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "pipelines",
}

func init() {

	Cmd.AddCommand(pipelines.Cmd)
}
