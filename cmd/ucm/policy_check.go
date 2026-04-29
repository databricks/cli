package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/cli/ucm/render"
	"github.com/spf13/cobra"
)

func newPolicyCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy-check",
		Short: "Run only the ucm validation mutators (tags, naming, required fields).",
		Long: `Run the subset of ucm validation mutators that are cheap enough for a
pre-commit hook. Unlike ` + "`ucm validate`" + `, which runs the full mutator chain,
policy-check only runs the validation rules (tag enforcement, naming,
required fields). No network I/O.`,
		Args: root.NoArgs,
	}

	var strict bool
	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.PolicyCheck(ctx, u)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		out := cmd.OutOrStdout()
		if root.OutputType(cmd) == flags.OutputText {
			if err1 := render.RenderDiagnosticsSummary(ctx, out, u); err1 != nil {
				return err1
			}
		}
		if root.OutputType(cmd) == flags.OutputJSON {
			if err1 := renderJsonOutput(cmd, u); err1 != nil {
				return err1
			}
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
