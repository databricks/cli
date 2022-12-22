package bundle

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// rootCmd represents the root command for the bundle subcommand.
var rootCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Databricks Application Bundles",
}

// ConfigureBundle loads the bundle configuration
// and configures it on the command's context.
func ConfigureBundle(cmd *cobra.Command, args []string) error {
	b, err := LoadAndSelectEnvironment(cmd)
	if err != nil {
		return err
	}
	cmd.SetContext(bundle.Context(cmd.Context(), b))
	return nil
}

func Load(cmd *cobra.Command) (*bundle.Bundle, error) {
	ctx := cmd.Context()
	b, err := bundle.LoadFromRoot()
	if err != nil {
		return nil, err
	}

	ms := mutator.DefaultMutators()
	err = bundle.Apply(ctx, b, ms)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func LoadAndSelectEnvironment(cmd *cobra.Command) (*bundle.Bundle, error) {
	b, err := Load(cmd)
	if err != nil {
		return nil, err
	}

	var m bundle.Mutator
	env := getEnvironment(cmd)
	if env == "" {
		m = mutator.SelectDefaultEnvironment()
	} else {
		m = mutator.SelectEnvironment(env)
	}

	err = bundle.Apply(cmd.Context(), b, []bundle.Mutator{m})
	if err != nil {
		return nil, err
	}

	return b, nil
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func environmentCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b, err := Load(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return maps.Keys(b.Config.Environments), cobra.ShellCompDirectiveDefault
}

func init() {
	// All bundle commands take an "environment" parameter.
	rootCmd.PersistentFlags().StringP("environment", "e", "", "Environment to use")

	rootCmd.RegisterFlagCompletionFunc("environment", environmentCompletion)

	// Add to top level root.
	root.RootCmd.AddCommand(rootCmd)
}
