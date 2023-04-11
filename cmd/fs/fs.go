package fs

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

// fsCmd represents the fs command
var fsCmd = &cobra.Command{
	Use:   "fs",
	Short: "Filesystem related commands",
	Long:  `Commands to do DBFS operations.`,
}

func init() {
	root.RootCmd.AddCommand(fsCmd)
}
