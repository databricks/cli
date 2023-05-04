package bundle

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

// rootCmd represents the root command for the bundle subcommand.
var rootCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Databricks Asset Bundles",
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

var variables []string

func init() {
	root.RootCmd.AddCommand(rootCmd)
	rootCmd.PersistentFlags().StringSliceVar(&variables, "var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)
}
