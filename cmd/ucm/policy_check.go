package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
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

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.PolicyCheck(ctx, u)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Policy check OK!")
		return nil
	}

	return cmd
}
