package ucm

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/render"
	"github.com/spf13/cobra"
)

// notDeployedURL is the literal rendered when a URL-bearing resource has no
// ID in the local tfstate. Matches the DAB wording at
// bundle/render/render_text_output.go so users reading both tools' output
// get a consistent signal.
const notDeployedURL = "(not deployed)"

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources declared by this ucm deployment.",
		Long: `Summarize the resources declared by this ucm deployment, grouped by kind,
with workspace URLs when a Workspace.Host is configured.

Mirrors ` + "`databricks bundle summary`" + `: loads the per-target
terraform.tfstate from the local cache to determine which resources have
actually been deployed. URL lines show the workspace console link for
resources present in state and ` + "`" + notDeployedURL + "`" + ` for resources declared in
ucm.yml but not yet applied. Run ` + "`ucm deploy`" + ` to realize declared intents.

Common invocations:
  databricks ucm summary                   # Text summary of the default target
  databricks ucm summary --target prod     # Summary of a named target
  databricks ucm summary -o json           # Emit the full config as JSON`,
		Args: root.NoArgs,
	}

	// forcePull is accepted for DAB parity but is a no-op today: summary reads
	// the local tfstate, not the remote workspace. Wiring a real state pull
	// belongs in a separate change.
	var forcePull bool
	var includeLocations bool
	var shouldShowFullConfig bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace (no-op today)")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	_ = cmd.Flags().MarkHidden("include-locations")
	cmd.Flags().BoolVar(&shouldShowFullConfig, "show-full-config", false, "Load and output the full ucm config")
	_ = cmd.Flags().MarkHidden("show-full-config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{
			InitIDs:          true,
			IncludeLocations: includeLocations,
		})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// --show-full-config short-circuits the grouped text renderer and
		// always emits the resolved config tree as JSON, regardless of -o.
		if shouldShowFullConfig {
			return renderJsonOutput(cmd, u)
		}

		return showSummary(cmd, u)
	}

	return cmd
}

func showSummary(cmd *cobra.Command, u *ucm.Ucm) error {
	if root.OutputType(cmd) == flags.OutputText {
		return render.RenderSummary(cmd.Context(), cmd.OutOrStdout(), u)
	}
	if root.OutputType(cmd) == flags.OutputJSON {
		return renderJsonOutput(cmd, u)
	}
	return nil
}
