package root

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root/bundleflag"
	"github.com/databricks/cli/cmd/root/profileflag"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

// configureProfile applies the profile flag to the bundle.
func configureProfile(cmd *cobra.Command, b *bundle.Bundle) diag.Diagnostics {
	profile, ok := profileflag.Value(cmd)
	if !ok || profile == "" {
		// If it's not set, use the environment variable.
		profile, ok = env.Lookup(cmd.Context(), "DATABRICKS_CONFIG_PROFILE")
	}

	if !ok || profile == "" {
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
	if target := bundleflag.Target(cmd); target == "" {
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
