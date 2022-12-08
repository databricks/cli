package workspace

import (
	"github.com/databricks/bricks/cmd/workspace/workspace"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace",
	Short: `The Workspace API allows you to list, import, export, and delete notebooks and folders.`,
	Long: `The Workspace API allows you to list, import, export, and delete notebooks and
  folders.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(workspace.Cmd)
}
