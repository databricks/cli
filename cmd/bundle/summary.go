package bundle

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
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
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			ReadState:        true,
			AlwaysPull:       forcePull,
			IncludeLocations: includeLocations,
			InitIDs:          true,
		})
		if err != nil {
			return err
		}
		return showSummary(cmd, b)
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
