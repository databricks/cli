package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

// rootCmd represents the root command for the bundle subcommand.
var rootCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Databricks Application Bundles",
}

// ConfigureBundle loads the bundle configuration
// and configures it on the command's context.
func ConfigureBundle(cmd *cobra.Command, args []string) error {
	ctx, err := bundle.ConfigureForEnvironment(cmd.Context(), getEnvironment(cmd))
	if err != nil {
		return err
	}

	cmd.SetContext(ctx)
	return nil
}

func AddCommand(cmd *cobra.Command) {
	// Most commands take an "environment" parameter, but init doesn't.
	// That's why it is NOT defined as a persistent flag on the root command.
	cmd.Flags().StringP("environment", "e", "", "Environment to use")
	rootCmd.AddCommand(cmd)
}

func init() {
	// Add to top level root.
	root.RootCmd.AddCommand(rootCmd)
}
