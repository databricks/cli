package fs

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

// fsCmd represents the fs command
var fsCmd = &cobra.Command{
	Use:   "fs",
	Short: "Filesystem related commands",
	Long:  `Commands to do DBFS operations.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("TODO")
	},
}

func init() {
	root.RootCmd.AddCommand(fsCmd)
}
