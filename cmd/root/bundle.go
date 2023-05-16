package root

import (
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const envName = "DATABRICKS_BUNDLE_ENV"

// getEnvironment returns the name of the environment to operate in.
func getEnvironment(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("environment")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	return os.Getenv(envName)
}

// loadBundle loads the bundle configuration and applies default mutators.
func loadBundle(cmd *cobra.Command, args []string, load func() (*bundle.Bundle, error)) (*bundle.Bundle, error) {
	b, err := load()
	if err != nil {
		return nil, err
	}

	// No bundle is fine in case of `TryConfigureBundle`.
	if b == nil {
		return nil, nil
	}

	ctx := cmd.Context()
	err = bundle.Apply(ctx, b, mutator.DefaultMutators())
	if err != nil {
		return nil, err
	}

	return b, nil
}

// configureBundle loads the bundle configuration and configures it on the command's context.
func configureBundle(cmd *cobra.Command, args []string, load func() (*bundle.Bundle, error)) error {
	b, err := loadBundle(cmd, args, load)
	if err != nil {
		return err
	}

	// No bundle is fine in case of `TryConfigureBundle`.
	if b == nil {
		return nil
	}

	var m bundle.Mutator
	env := getEnvironment(cmd)
	if env == "" {
		m = mutator.SelectDefaultEnvironment()
	} else {
		m = mutator.SelectEnvironment(env)
	}

	ctx := cmd.Context()
	err = bundle.Apply(ctx, b, []bundle.Mutator{m})
	if err != nil {
		return err
	}

	cmd.SetContext(bundle.Context(ctx, b))
	return nil
}

// MustConfigureBundle configures a bundle on the command context.
func MustConfigureBundle(cmd *cobra.Command, args []string) error {
	return configureBundle(cmd, args, bundle.MustLoad)
}

// TryConfigureBundle configures a bundle on the command context
// if there is one, but doesn't fail if there isn't one.
func TryConfigureBundle(cmd *cobra.Command, args []string) error {
	return configureBundle(cmd, args, bundle.TryLoad)
}

// environmentCompletion executes to autocomplete the argument to the environment flag.
func environmentCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b, err := loadBundle(cmd, args, bundle.MustLoad)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return maps.Keys(b.Config.Environments), cobra.ShellCompDirectiveDefault
}

func init() {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	RootCmd.PersistentFlags().StringP("environment", "e", "", "bundle environment to use (if applicable)")
	RootCmd.RegisterFlagCompletionFunc("environment", environmentCompletion)
}
