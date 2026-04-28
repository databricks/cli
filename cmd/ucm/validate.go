package ucm

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/render"
	"github.com/spf13/cobra"
)

func renderJsonOutput(cmd *cobra.Command, u *ucm.Ucm) error {
	if u == nil {
		return nil
	}
	buf, err := json.MarshalIndent(u.Config.Value().AsAny(), "", "  ")
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
		Short: "Validate ucm.yml for errors, warnings, policy violations.",
		Long: `Validate ucm configuration for errors, warnings and policy violations.

Runs the full ucm mutator chain, including tag-validation rules, against the
selected target. Useful as a CI gate before ` + "`ucm deploy`" + `.

Common invocations:
  databricks ucm validate                  # Validate default target
  databricks ucm validate --target prod    # Validate a specific target
  databricks ucm validate --strict         # Fail on warnings too
  databricks ucm validate -o json          # Emit the full config as JSON`,
		Args: root.NoArgs,
		// Diagnostics are already surfaced; don't spam usage on validation fail.
		SilenceUsage: true,
	}

	var strict bool
	var includeLocations bool
	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	_ = cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{
			Validate:         true,
			IncludeLocations: includeLocations,
		})
		ctx := cmd.Context()

		if err != nil && err != root.ErrAlreadyPrinted {
			logdiag.LogError(ctx, err)
			err = root.ErrAlreadyPrinted
		}

		// output before checking the error on purpose

		if root.OutputType(cmd) == flags.OutputText {
			err1 := render.RenderDiagnosticsSummary(ctx, cmd.OutOrStdout(), u)
			if err1 != nil {
				return err1
			}
		}

		if root.OutputType(cmd) == flags.OutputJSON {
			err1 := renderJsonOutput(cmd, u)
			if err1 != nil {
				return err1
			}
		}

		// In strict mode, treat warnings as errors.
		numWarnings := logdiag.NumWarnings(ctx)
		if err == nil && strict && numWarnings > 0 {
			prefix := ""
			if numWarnings == 1 {
				prefix = "1 warning was found"
			} else {
				prefix = fmt.Sprintf("%d warnings were found", numWarnings)
			}
			return fmt.Errorf("%s. Warnings are not allowed in strict mode", prefix)
		}

		return err
	}

	return cmd
}
