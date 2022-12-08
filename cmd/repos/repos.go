package repos

import (
	"github.com/databricks/bricks/cmd/repos/repos"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "repos",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(repos.Cmd)
}
