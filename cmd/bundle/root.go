package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/mutator"
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
	ctx := cmd.Context()
	b, err := bundle.LoadFromRoot()
	if err != nil {
		return err
	}

	ms := mutator.DefaultMutators()
	env := getEnvironment(cmd)
	if env == "" {
		ms = append(ms, mutator.SelectDefaultEnvironment())
	} else {
		ms = append(ms, mutator.SelectEnvironment(env))
	}

	err = bundle.Apply(ctx, b, ms)
	if err != nil {
		return err
	}

	cmd.SetContext(bundle.Context(ctx, b))
	return nil
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func init() {
	// All bundle commands take an "environment" parameter.
	rootCmd.PersistentFlags().StringP("environment", "e", "", "Environment to use")
	// Add to top level root.
	root.RootCmd.AddCommand(rootCmd)
}
