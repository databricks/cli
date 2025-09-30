package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) {
	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.InitializeVariables(variables)
		if err != nil {
			logdiag.LogError(ctx, err)
		}
	})
}

func ConfigureBundleWithVariables(cmd *cobra.Command) *bundle.Bundle {
	// Load bundle config and apply target
	b := root.MustConfigureBundle(cmd)
	ctx := cmd.Context()
	if logdiag.HasError(ctx) {
		return b
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(cmd, b, variables)

	return b
}
