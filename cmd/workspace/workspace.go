package workspace

import (
	"github.com/databricks/bricks/cmd/workspace/workspace"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "workspace",
}

func init() {

	Cmd.AddCommand(workspace.Cmd)
}
