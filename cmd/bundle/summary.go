package bundle

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources deployed by this bundle",
		Long: `Summarize resources deployed by this bundle with their workspace URLs.
Useful after deployment to see what was created and where to find it.`,
		Args: root.NoArgs,
	}

	var forcePull bool
	var includeLocations bool
	var shouldShowFullConfig bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")
	cmd.Flags().BoolVar(&shouldShowFullConfig, "show-full-config", false, "Load and output the full bundle config")
	cmd.Flags().MarkHidden("show-full-config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var err error
		if shouldShowFullConfig {
			ctx := logdiag.InitContext(cmd.Context())
			cmd.SetContext(ctx)
			logdiag.SetSeverity(ctx, diag.Warning)

			err = showFullConfig(ctx, cmd)
			if err != nil {
				return err
			}
		} else {
			b, err := utils.ProcessBundle(cmd, &utils.ProcessOptions{
				ReadState:        true,
				IncludeLocations: includeLocations,
				InitIDs:          true,
			})
			if err != nil {
				return err
			}
			err = showSummary(cmd, b)
			if err != nil {
				return err
			}
		}

		if logdiag.HasError(cmd.Context()) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}

func showFullConfig(ctx context.Context, cmd *cobra.Command) error {
	// call `MustLoad` directly instead of `MustConfigureBundle` because the latter does
	// validation that we're not interested in here
	b := bundle.MustLoad(ctx)
	if b == nil || logdiag.HasError(ctx) {
		return nil
	}

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return nil
	}

	err := renderJsonOutput(cmd, b)
	if err != nil {
		return err
	}

	return nil
}

func showSummary(cmd *cobra.Command, b *bundle.Bundle) error {
	if root.OutputType(cmd) == flags.OutputText {
		return render.RenderSummary(cmd.Context(), cmd.OutOrStdout(), b)
	}
	if root.OutputType(cmd) == flags.OutputJSON {
		return renderJsonOutput(cmd, b)
	}
	return nil
}
