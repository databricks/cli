package workspaceconf

import (
	workspace_conf "github.com/databricks/bricks/cmd/workspaceconf/workspace-conf"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspaceconf",
	Short: `This API allows updating known workspace settings for advanced users.`,
	Long:  `This API allows updating known workspace settings for advanced users.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(workspace_conf.Cmd)
}
