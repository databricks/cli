package bundleflag

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

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
