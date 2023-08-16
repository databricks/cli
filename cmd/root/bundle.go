package root

import (
	"context"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const envName = "DATABRICKS_BUNDLE_ENV"

// getTarget returns the name of the target to operate in.
func getTarget(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("target")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	oldFlag := cmd.Flag("environment")
	if oldFlag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	return os.Getenv(envName)
}

func getProfile(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("profile")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	return os.Getenv("DATABRICKS_CONFIG_PROFILE")
}

// loadBundle loads the bundle configuration and applies default mutators.
func loadBundle(cmd *cobra.Command, args []string, load func(ctx context.Context) (*bundle.Bundle, error)) (*bundle.Bundle, error) {
	ctx := cmd.Context()
	b, err := load(ctx)
	if err != nil {
		return nil, err
	}

	// No bundle is fine in case of `TryConfigureBundle`.
	if b == nil {
		return nil, nil
	}

	profile := getProfile(cmd)
	if profile != "" {
		b.Config.Workspace.Profile = profile
	}

	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	if err != nil {
		return nil, err
	}

	return b, nil
}

// configureBundle loads the bundle configuration and configures it on the command's context.
func configureBundle(cmd *cobra.Command, args []string, load func(ctx context.Context) (*bundle.Bundle, error)) error {
	b, err := loadBundle(cmd, args, load)
	if err != nil {
		return err
	}

	// No bundle is fine in case of `TryConfigureBundle`.
	if b == nil {
		return nil
	}

	var m bundle.Mutator
	env := getTarget(cmd)
	if env == "" {
		m = mutator.SelectDefaultTarget()
	} else {
		m = mutator.SelectTarget(env)
	}

	ctx := cmd.Context()
	err = bundle.Apply(ctx, b, m)
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

// targetCompletion executes to autocomplete the argument to the target flag.
func targetCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b, err := loadBundle(cmd, args, bundle.MustLoad)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return maps.Keys(b.Config.Targets), cobra.ShellCompDirectiveDefault
}

func initTargetFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "target" parameter.
	cmd.PersistentFlags().String("target", "", "bundle target to use (if applicable)")
	cmd.RegisterFlagCompletionFunc("target", targetCompletion)
}

// DEPRECATED flag
func initEnvironmentFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	cmd.PersistentFlags().StringP("environment", "e", "", "bundle target to use (if applicable)")
	cmd.PersistentFlags().MarkDeprecated("environment", "use --target flag instead")
	cmd.RegisterFlagCompletionFunc("environment", targetCompletion)
}
