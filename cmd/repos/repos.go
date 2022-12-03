package repos

import (
	"github.com/databricks/bricks/cmd/repos/repos"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "repos",
}

func init() {

	Cmd.AddCommand(repos.Cmd)
}
