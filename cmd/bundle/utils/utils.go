package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func ConfigureBundleWithVariables(cmd *cobra.Command, args []string) error {
	// Load bundle config and apply target
	err := root.MustConfigureBundle(cmd, args)
	if err != nil {
		return err
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return err
	}

	// Initialize variables by assigning them values passed as command line flags
	b := bundle.Get(cmd.Context())
	return bundle.ApplyFunc(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) error {
		return b.Config.InitializeVariables(variables)
	})
}
