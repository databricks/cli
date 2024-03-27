package root

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/diag"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

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
		value = oldFlag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	target, _ := env.Target(cmd.Context())
	return target
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
	return envlib.Get(cmd.Context(), "DATABRICKS_CONFIG_PROFILE")
}

// configureProfile applies the profile flag to the bundle.
func configureProfile(cmd *cobra.Command, b *bundle.Bundle) diag.Diagnostics {
	profile := getProfile(cmd)
	if profile == "" {
		return nil
	}

	return bundle.ApplyFunc(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.Profile = profile
		return nil
	})
}

// configureBundle applies basic mutators to the bundle configures it on the command's context.
func configureBundle(cmd *cobra.Command, b *bundle.Bundle) (*bundle.Bundle, diag.Diagnostics) {
	var m bundle.Mutator
	if target := getTarget(cmd); target == "" {
		m = phases.LoadDefaultTarget()
	} else {
		m = phases.LoadNamedTarget(target)
	}

	// Load bundle and select target.
	ctx := cmd.Context()
	diags := bundle.Apply(ctx, b, m)
	if diags.HasError() {
		return nil, diags
	}

	// Configure the workspace profile if the flag has been set.
	diags = diags.Extend(configureProfile(cmd, b))
	if diags.HasError() {
		return nil, diags
	}

	return b, diags
}

// MustConfigureBundle configures a bundle on the command context.
func MustConfigureBundle(cmd *cobra.Command) (*bundle.Bundle, diag.Diagnostics) {
	b, err := bundle.MustLoad(cmd.Context())
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return configureBundle(cmd, b)
}

// TryConfigureBundle configures a bundle on the command context
// if there is one, but doesn't fail if there isn't one.
func TryConfigureBundle(cmd *cobra.Command) (*bundle.Bundle, diag.Diagnostics) {
	b, err := bundle.TryLoad(cmd.Context())
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// No bundle is fine in this case.
	if b == nil {
		return nil, nil
	}

	return configureBundle(cmd, b)
}

// targetCompletion executes to autocomplete the argument to the target flag.
func targetCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	b, err := bundle.MustLoad(ctx)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	// Load bundle but don't select a target (we're completing those).
	diags := bundle.Apply(ctx, b, phases.Load())
	if err := diags.Error(); err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return maps.Keys(b.Config.Targets), cobra.ShellCompDirectiveDefault
}

func initTargetFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "target" parameter.
	cmd.PersistentFlags().StringP("target", "t", "", "bundle target to use (if applicable)")
	cmd.RegisterFlagCompletionFunc("target", targetCompletion)
}

// DEPRECATED flag
func initEnvironmentFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	cmd.PersistentFlags().StringP("environment", "e", "", "bundle target to use (if applicable)")
	cmd.PersistentFlags().MarkDeprecated("environment", "use --target flag instead")
	cmd.RegisterFlagCompletionFunc("environment", targetCompletion)
}
