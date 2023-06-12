package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// cpCmd represents the fs cp command
var cpCmd = &cobra.Command{
	Use:     "cp SOURCE_PATH TARGET_PATH",
	Short:   "Copy files to and from DBFS.",
	Long:    `TODO`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {

	},
}

func init() {
	fsCmd.AddCommand(cpCmd)
}
