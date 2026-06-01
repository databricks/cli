package bundle

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
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
	var selectResources []string
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")
	cmd.Flags().StringArrayVar(&selectResources, "select", nil, "Show only the specified resource (e.g. 'my_job' or 'jobs.my_job'). Can be repeated.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			ReadState:        true,
			AlwaysPull:       forcePull,
			IncludeLocations: includeLocations,
			InitIDs:          true,
			InitFunc: func(b *bundle.Bundle) {
				b.Select = selectResources
			},
		})
		if err != nil {
			return err
		}
		err = showSummary(cmd, b)
		if err != nil {
			return err
		}

		if logdiag.HasError(cmd.Context()) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
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
