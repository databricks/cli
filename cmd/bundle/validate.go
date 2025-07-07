package bundle

import (
	"encoding/json"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
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
		Args:  root.NoArgs,
	}

	var includeLocations bool
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)

		if b == nil {
			if err := diags.Error(); err != nil {
				return diags.Error()
			} else {
				return errors.New("invariant failed: returned bundle is nil")
			}
		}

		if !diags.HasError() {
			diags = diags.Extend(phases.Initialize(ctx, b))
		}

		if !diags.HasError() {
			diags = diags.Extend(validate.Validate(ctx, b))
		}

		// Include location information in the output if the flag is set.
		if includeLocations {
			diags = diags.Extend(bundle.Apply(ctx, b, mutator.PopulateLocations()))
		}

		if root.OutputType(cmd) == flags.OutputText {
			err := render.RenderDiagnostics(cmd.OutOrStdout(), b, diags, render.RenderOptions{RenderSummaryTable: true})
			if err != nil {
				return err
			}
		}
		if root.OutputType(cmd) == flags.OutputJSON {
			err := render.RenderDiagnostics(cmd.ErrOrStderr(), b, diags, render.RenderOptions{RenderSummaryTable: false})
			if err != nil {
				return err
			}
			err = renderJsonOutput(cmd, b)
			if err != nil {
				return err
			}
		}
		if diags.HasError() {
			return root.ErrAlreadyPrinted
		}
		return nil
	}

	return cmd
}
