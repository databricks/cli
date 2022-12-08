package libraries

import (
	"github.com/databricks/bricks/cmd/libraries/libraries"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "libraries",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(libraries.Cmd)
}
