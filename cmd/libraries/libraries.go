package libraries

import (
	"github.com/databricks/bricks/cmd/libraries/libraries"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "libraries",
	Short: `The Libraries API allows you to install and uninstall libraries and get the status of libraries on a cluster.`,
	Long: `The Libraries API allows you to install and uninstall libraries and get the
  status of libraries on a cluster.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(libraries.Cmd)
}
