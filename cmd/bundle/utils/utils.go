package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/spf13/cobra"
)

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) diag.Diagnostics {
	return bundle.ApplyFunc(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.InitializeVariables(variables)
		return diag.FromErr(err)
	})
}

func ConfigureBundleWithVariables(cmd *cobra.Command) (*bundle.Bundle, diag.Diagnostics) {
	// Load bundle config and apply target
	b, diags := root.MustConfigureBundle(cmd)
	if diags.HasError() {
		return b, diags
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return b, diag.FromErr(err)
	}

	// Initialize variables by assigning them values passed as command line flags
	diags = diags.Extend(configureVariables(cmd, b, variables))

	return b, diags
}
