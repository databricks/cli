package globalinitscripts

import (
	global_init_scripts "github.com/databricks/bricks/cmd/globalinitscripts/global-init-scripts"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "globalinitscripts",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(global_init_scripts.Cmd)
}
