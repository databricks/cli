package ucm

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/phases"
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

		plan := result.Plan
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
			renderPlanText(out, plan)
			return nil
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

// renderPlanText prints the DAB-style per-resource action list followed by
// the `Plan: N to add, ...` tally. Mirrors cmd/bundle/plan.go so ucm plan
// output is byte-identical to bundle plan for the resources ucm models.
func renderPlanText(out io.Writer, plan *deployplan.Plan) {
	createCount, updateCount, deleteCount, unchangedCount := 0, 0, 0, 0
	for _, change := range plan.GetActions() {
		switch change.ActionType {
		case deployplan.Create:
			createCount++
		case deployplan.Update, deployplan.UpdateWithID, deployplan.Resize:
			updateCount++
		case deployplan.Delete:
			deleteCount++
		case deployplan.Recreate:
			deleteCount++
			createCount++
		case deployplan.Skip, deployplan.Undefined:
			unchangedCount++
		}
	}

	totalChanges := createCount + updateCount + deleteCount
	if totalChanges > 0 {
		for _, action := range plan.GetActions() {
			if action.ActionType == deployplan.Skip {
				continue
			}
			key := strings.TrimPrefix(action.ResourceKey, "resources.")
			fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Plan: %d to add, %d to change, %d to delete, %d unchanged\n", createCount, updateCount, deleteCount, unchangedCount)
}
