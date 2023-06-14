package bundle

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// rootCmd represents the root command for the bundle subcommand.
var rootCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Databricks Application Bundles",
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

var variables []string

func init() {
	root.RootCmd.AddCommand(rootCmd)
	AddVariableFlag(rootCmd)
}
