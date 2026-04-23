package ucm

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/cli/ucm/render"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview the changes ucm deploy would make.",
		Long: `Preview the changes ucm deploy would make.

Runs the initialize → build → terraform init → terraform plan sequence and
prints a DAB-style action list plus the add/change/delete/unchanged tally.
No state is mutated and no remote resources are touched.

Common invocations:
  databricks ucm plan                   # Plan against the default target
  databricks ucm plan --target prod     # Plan against a specific target
  databricks ucm plan -o json           # Emit the structured plan as JSON`,
		Args:    root.NoArgs,
		PreRunE: utils.MustWorkspaceClient,
	}

	var forceLock bool
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

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
		opts.ForceLock = forceLock

		outcome := phases.Plan(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}
		if outcome == nil {
			return root.ErrAlreadyPrinted
		}

		plan := outcome.Plan
		if plan == nil {
			plan = deployplan.NewPlanTerraform()
		}

		out := cmd.OutOrStdout()
		switch planOutputType(cmd) {
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			return nil
		default:
			return render.RenderPlan(out, plan)
		}
	}

	return cmd
}

// planOutputType returns the configured -o value, defaulting to OutputText
// when the flag is not wired (e.g. in standalone unit tests that don't go
// through root.New). root.OutputType would panic in that case.
func planOutputType(cmd *cobra.Command) flags.Output {
	if cmd.Flag("output") == nil {
		return flags.OutputText
	}
	return root.OutputType(cmd)
}
