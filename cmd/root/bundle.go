package root

import (
	"os"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/spf13/cobra"
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

// configureBundle loads the bundle configuration and configures it on the command's context.
func configureBundle(cmd *cobra.Command, args []string, load func() (*bundle.Bundle, error)) error {
	b, err := load()
	if err != nil {
		return err
	}

	// No bundle is fine in case of `TryConfigureBundle`.
	if b == nil {
		return nil
	}

	ms := mutator.DefaultMutators()
	env := getEnvironment(cmd)
	if env == "" {
		ms = append(ms, mutator.SelectDefaultEnvironment())
	} else {
		ms = append(ms, mutator.SelectEnvironment(env))
	}

	ctx := cmd.Context()
	err = bundle.Apply(ctx, b, ms)
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

func init() {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	RootCmd.PersistentFlags().StringP("environment", "e", "", "bundle environment to use (if applicable)")
}
