package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview the changes ucm deploy would make.",
		Long: `Preview the changes ucm deploy would make.

Runs the initialize → build → terraform init → terraform plan sequence and
prints a one-line summary. No state is mutated and no remote resources are
touched.

Common invocations:
  databricks ucm plan                   # Plan against the default target
  databricks ucm plan --target prod     # Plan against a specific target`,
		Args:    root.NoArgs,
		PreRunE: utils.MustWorkspaceClient,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve deploy options: %w", err)
		}

		result := phases.Plan(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}
		if result == nil {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintln(cmd.OutOrStdout(), result.Summary)
		return nil
	}

	return cmd
}
