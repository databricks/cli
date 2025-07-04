// Copied from cmd/root/bundle.go and adapted for pipelines use.
package root

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

// targetCompletion executes to autocomplete the argument to the target flag.
func targetCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	b := bundle.MustLoad(ctx)
	if b == nil || logdiag.HasError(ctx) {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load project but don't select a target (we're completing those).
	phases.Load(ctx, b)
	if logdiag.HasError(ctx) {
		return nil, cobra.ShellCompDirectiveError
	}

	return maps.Keys(b.Config.Targets), cobra.ShellCompDirectiveDefault
}

func initTargetFlag(cmd *cobra.Command) {
	// To operate in the context of a project, all commands must take an "target" parameter.
	cmd.PersistentFlags().StringP("target", "t", "", "project target to use (if applicable)")
	cmd.RegisterFlagCompletionFunc("target", targetCompletion)
}
