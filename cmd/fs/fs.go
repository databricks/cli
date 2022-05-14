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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
