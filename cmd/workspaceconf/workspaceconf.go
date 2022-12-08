package workspaceconf

import (
	workspace_conf "github.com/databricks/bricks/cmd/workspaceconf/workspace-conf"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "workspaceconf",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(workspace_conf.Cmd)
}
