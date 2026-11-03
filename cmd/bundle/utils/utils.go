package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

// SetForceLock applies the --force-lock flag to the bundle configuration.
//
// The flag must only override bundle.deployment.lock.force when the user passed
// it explicitly. InitFunc runs after the configuration is loaded, so assigning
// unconditionally would let the flag's default of false clobber a force: true
// set in databricks.yml. Commands that share this code path without registering
// the flag (bundle generate, which binds with forcing disabled) leave the
// configured value untouched.
func SetForceLock(cmd *cobra.Command, b *bundle.Bundle, forceLock bool) {
	flag := cmd.Flags().Lookup("force-lock")
	if flag == nil || !flag.Changed {
		return
	}

	b.Config.Bundle.Deployment.Lock.Force = forceLock
}

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) {
	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.InitializeVariables(variables)
		if err != nil {
			logdiag.LogError(ctx, err)
		}
	})
}
