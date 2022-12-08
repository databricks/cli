package globalinitscripts

import (
	global_init_scripts "github.com/databricks/bricks/cmd/globalinitscripts/global-init-scripts"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "globalinitscripts",
	Short: `The Global Init Scripts API enables Workspace administrators to configure global initialization scripts for their workspace.`,
	Long: `The Global Init Scripts API enables Workspace administrators to configure
  global initialization scripts for their workspace.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(global_init_scripts.Cmd)
}
