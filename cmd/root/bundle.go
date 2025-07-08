package root

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdctx"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// getTarget returns the name of the target to operate in.
func getTarget(cmd *cobra.Command) (value string) {
	target, isFlagSet := targetFlagValue(cmd)
	if isFlagSet {
		return target
	}

	// If it's not set, use the environment variable.
	target, _ = env.Target(cmd.Context())
	return target
}

func targetFlagValue(cmd *cobra.Command) (string, bool) {
	// The command line flag takes precedence.
	flag := cmd.Flag("target")
	if flag != nil {
		value := flag.Value.String()
		if value != "" {
			return value, true
		}
	}

	oldFlag := cmd.Flag("environment")
	if oldFlag != nil {
		value := oldFlag.Value.String()
		if value != "" {
			return value, true
		}
	}

	return "", false
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
func configureProfile(cmd *cobra.Command, b *bundle.Bundle) {
	profile := getProfile(cmd)
	if profile == "" {
		return
	}

	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.Profile = profile
	})
}

// configureBundle loads the bundle configuration and configures flag values, if any.
func configureBundle(cmd *cobra.Command, b *bundle.Bundle) {
	// Load bundle and select target.
	ctx := cmd.Context()
	if target := getTarget(cmd); target == "" {
		phases.LoadDefaultTarget(ctx, b)
	} else {
		phases.LoadNamedTarget(ctx, b, target)
	}

	if logdiag.HasError(ctx) {
		return
	}

	// Configure the workspace profile if the flag has been set.
	configureProfile(cmd, b)

	// Set the auth configuration in the command context. This can be used
	// downstream to initialize a API client.
	//
	// Note that just initializing a workspace client and loading auth configuration
	// is a fast operation. It does not perform network I/O or invoke processes (for example the Azure CLI).
	client, err := b.WorkspaceClientE()
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	ctx = cmdctx.SetConfigUsed(ctx, client.Config)
	cmd.SetContext(ctx)
}

// MustConfigureBundle configures a bundle on the command context.
func MustConfigureBundle(cmd *cobra.Command) *bundle.Bundle {
	// A bundle may be configured on the context when testing.
	// If it is, return it immediately.
	b := bundle.GetOrNil(cmd.Context())
	if b != nil {
		return b
	}

	b = bundle.MustLoad(cmd.Context())
	if b != nil {
		configureBundle(cmd, b)
	}
	return b
}

// TryConfigureBundle configures a bundle on the command context
// if there is one, but doesn't fail if there isn't one.
func TryConfigureBundle(cmd *cobra.Command) *bundle.Bundle {
	// A bundle may be configured on the context when testing.
	// If it is, return it immediately.
	b := bundle.GetOrNil(cmd.Context())
	if b != nil {
		return b
	}

	ctx := cmd.Context()
	b = bundle.TryLoad(ctx)
	// No bundle is fine in this case.
	if b == nil || logdiag.HasError(ctx) {
		return nil
	}

	configureBundle(cmd, b)
	return b
}

// targetCompletion executes to autocomplete the argument to the target flag.
func targetCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	b := bundle.MustLoad(ctx)
	if b == nil || logdiag.HasError(ctx) {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load bundle but don't select a target (we're completing those).
	phases.Load(ctx, b)
	if logdiag.HasError(ctx) {
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
