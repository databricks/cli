package bundle

import (
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func renderJsonOutput(cmd *cobra.Command, b *bundle.Bundle) error {
	if b == nil {
		return nil
	}
	buf, err := json.MarshalIndent(b.Config.Value().AsAny(), "", "  ")
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = out.Write(buf)
	_, _ = out.Write([]byte{'\n'})
	return nil
}

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long: `Validate bundle configuration for errors, warnings and recommendations.

Run validate before deploy to catch configuration issues early:
  databricks bundle validate                  # Validate default target
  databricks bundle validate --target prod    # Validate specific target

Validation checks the configuration syntax and schema, permissions etc.

Please run this command before deploying to ensure configuration quality.`,
		Args: root.NoArgs,
	}

	var includeLocations bool
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			Validate:         true,
			IncludeLocations: includeLocations,
		})
		ctx := cmd.Context()

		// output before checking the error on purpose

		if root.OutputType(cmd) == flags.OutputText {
			err1 := render.RenderDiagnosticsSummary(ctx, cmd.OutOrStdout(), b)
			if err1 != nil {
				return err1
			}
		}

		if root.OutputType(cmd) == flags.OutputJSON {
			err1 := renderJsonOutput(cmd, b)
			if err1 != nil {
				return err1
			}
		}

		return err
	}

	return cmd
}
