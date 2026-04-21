package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
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
  databricks ucm validate --strict         # Fail on warnings too`,
		Args: root.NoArgs,
	}

	var strict bool
	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		utils.ProcessUcm(cmd, utils.ProcessOptions{Validate: true})
		ctx := cmd.Context()

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

		fmt.Fprintln(cmd.OutOrStdout(), "Validation OK!")
		return nil
	}

	return cmd
}
