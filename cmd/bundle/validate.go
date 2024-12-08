package bundle

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func renderJsonOutput(cmd *cobra.Command, b *bundle.Bundle, diags diag.Diagnostics) error {
	buf, err := json.MarshalIndent(b.Config.Value().AsAny(), "", "  ")
	if err != nil {
		return err
	}
	cmd.OutOrStdout().Write(buf)
	return diags.Error()
}

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)

		if b == nil {
			if err := diags.Error(); err != nil {
				return diags.Error()
			} else {
				return fmt.Errorf("invariant failed: returned bundle is nil")
			}
		}

		if !diags.HasError() {
			diags = diags.Extend(bundle.Apply(ctx, b, phases.Initialize()))
		}

		if !diags.HasError() {
			diags = diags.Extend(bundle.Apply(ctx, b, validate.Validate()))
		}

		switch root.OutputType(cmd) {
		case flags.OutputText:
			renderOpts := render.RenderOptions{RenderSummaryTable: true}
			err := render.RenderDiagnostics(cmd.OutOrStdout(), b, diags, renderOpts)
			if err != nil {
				return fmt.Errorf("failed to render output: %w", err)
			}

			if diags.HasError() {
				return root.ErrAlreadyPrinted
			}

			return nil
		case flags.OutputJSON:
			return renderJsonOutput(cmd, b, diags)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}
	}

	return cmd
}
