package ucm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{Validate: true})
		ctx := cmd.Context()
		if err != nil {
			return err
		}

		if includeLocations && u != nil && !logdiag.HasError(ctx) {
			ucm.ApplyContext(ctx, u, mutator.PopulateLocations())
		}

		out := cmd.OutOrStdout()
		output := validateOutputType(cmd)

		// Emit output before returning on error so users see the summary or
		// the (partial) config tree regardless.
		if output == flags.OutputJSON {
			if err := renderValidateJSON(out, u); err != nil {
				return err
			}
		} else {
			if u != nil {
				renderSummaryHeader(out, u)
			}
			writeValidateTrailer(ctx, out)
		}

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		numWarnings := logdiag.NumWarnings(ctx)
		if strict && numWarnings > 0 {
			noun := "warning"
			if numWarnings != 1 {
				noun = "warnings"
			}
			return fmt.Errorf("%d %s found. Warnings are not allowed in strict mode", numWarnings, noun)
		}

		return nil
	}

	return cmd
}

// validateOutputType returns the configured -o value, defaulting to OutputText
// when the flag is not wired (e.g. in standalone unit tests that don't go
// through root.New). root.OutputType would panic in that case.
func validateOutputType(cmd *cobra.Command) flags.Output {
	if cmd.Flag("output") == nil {
		return flags.OutputText
	}
	return root.OutputType(cmd)
}

// renderValidateJSON emits the loaded ucm config tree as indented JSON.
func renderValidateJSON(out io.Writer, u *ucm.Ucm) error {
	if u == nil {
		return nil
	}
	buf, err := json.MarshalIndent(u.Config.Value().AsAny(), "", "  ")
	if err != nil {
		return err
	}
	_, _ = out.Write(buf)
	_, _ = out.Write([]byte{'\n'})
	return nil
}

// writeValidateTrailer prints the DAB-style "Found X errors / Y warnings"
// summary, or "Validation OK!" when no diagnostics were recorded.
func writeValidateTrailer(ctx context.Context, out io.Writer) {
	info := logdiag.Copy(ctx)
	var parts []string
	if info.Errors > 0 {
		parts = append(parts, color.RedString(pluralize(info.Errors, "error", "errors")))
	}
	if info.Warnings > 0 {
		parts = append(parts, color.YellowString(pluralize(info.Warnings, "warning", "warnings")))
	}
	if info.Recommendations > 0 {
		parts = append(parts, color.BlueString(pluralize(info.Recommendations, "recommendation", "recommendations")))
	}
	switch len(parts) {
	case 0:
		fmt.Fprint(out, color.GreenString("Validation OK!\n"))
	case 1:
		fmt.Fprintf(out, "Found %s\n", parts[0])
	case 2:
		fmt.Fprintf(out, "Found %s and %s\n", parts[0], parts[1])
	default:
		first := strings.Join(parts[:len(parts)-1], ", ")
		fmt.Fprintf(out, "Found %s, and %s\n", first, parts[len(parts)-1])
	}
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
