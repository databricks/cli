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
	"github.com/databricks/cli/libs/logdiag"
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
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := prepareBundleForValidate(cmd, includeLocations)

		if b == nil {
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			} else {
				return errors.New("invariant failed: returned bundle is nil")
			}
		}

		if root.OutputType(cmd) == flags.OutputText {
			err := render.RenderDiagnosticsSummary(ctx, cmd.OutOrStdout(), b)
			if err != nil {
				return err
			}
		}

		if root.OutputType(cmd) == flags.OutputJSON {
			err := renderJsonOutput(cmd, b)
			if err != nil {
				return err
			}
		}

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}

func prepareBundleForValidate(cmd *cobra.Command, includeLocations bool) *bundle.Bundle {
	b := utils.ConfigureBundleWithVariables(cmd)
	ctx := cmd.Context()

	if b == nil || logdiag.HasError(ctx) {
		return b
	}

	phases.Initialize(ctx, b)

	if logdiag.HasError(ctx) {
		return b
	}

	validate.Validate(ctx, b)

	if logdiag.HasError(ctx) {
		return b
	}

	// Include location information in the output if the flag is set.
	if includeLocations {
		bundle.ApplyContext(ctx, b, mutator.PopulateLocations())
	}

	return b
}
