package bundle

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func renderJsonOutput(cmd *cobra.Command, b *bundle.Bundle) error {
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

		return renderBundle(cmd, b, diags, false)
	}

	return cmd
}

// This function is used to render results both for "bundle validate" and "bundle summary".
// In JSON mode, there is no difference in rendering between these two (but there is a difference in how we prepare the bundle).
// In non-JSON mode both "bundle validate" and "bundle summary" will print diagnostics to stderr but "bundle validate"
// will also print "summary" message via RenderSummaryTable option.
func renderBundle(cmd *cobra.Command, b *bundle.Bundle, diags diag.Diagnostics, withBundleSummary bool) error {
	ctx := cmd.Context()
	switch root.OutputType(cmd) {
	case flags.OutputText:
		// Confusingly RenderSummaryTable relates to "Validation OK" and related messages, it has nothing
		// to do with "bundle summary" command and we don't want to show it in bundle summary command.
		renderOpts := render.RenderOptions{RenderSummaryTable: !withBundleSummary}
		err1 := render.RenderDiagnostics(cmd.OutOrStdout(), b, diags, renderOpts)
		if b != nil && withBundleSummary {
			// Now RenderSummary actually related to "bundle summary"
			err2 := render.RenderSummary(ctx, cmd.OutOrStdout(), b)
			if err2 != nil {
				return err2
			}
		}

		if err1 != nil {
			return err1
		}

		if diags.HasError() {
			return root.ErrAlreadyPrinted
		}

		return nil
	case flags.OutputJSON:
		renderOpts := render.RenderOptions{RenderSummaryTable: false}
		err1 := render.RenderDiagnostics(cmd.ErrOrStderr(), b, diags, renderOpts)
		err2 := renderJsonOutput(cmd, b)

		if err2 != nil {
			return err2
		}

		if err1 != nil {
			return err1
		}

		if diags.HasError() {
			return root.ErrAlreadyPrinted
		}

		return nil
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}
