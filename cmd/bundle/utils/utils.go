package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/spf13/cobra"
)

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) error {
	var initErr error
	if err := bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		initErr = b.Config.InitializeVariables(variables)
	}); err != nil {
		return err
	}
	return initErr
}
