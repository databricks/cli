// Package render formats ucm plan/apply output for the CLI. Mirrors the
// structure of bundle/render so future work (apply rendering, coloring,
// grouping) has a place to land without another round of extraction.
package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/ucm/deployplan"
)

// PlanSummary returns the legacy one-liner that ucm/phases emits as the
// PlanOutcome.Summary. Centralized here so both engines call into the
// same render-package helper.
func PlanSummary(hasChanges bool) string {
	if hasChanges {
		return "plan has changes"
	}
	return "no changes"
}

// RenderPlan prints the DAB-style per-resource action list followed by the
// `Plan: N to add, ...` tally. Mirrors cmd/bundle/plan.go so ucm plan output
// is byte-identical to bundle plan for the resources ucm models.
func RenderPlan(w io.Writer, plan *deployplan.Plan) error {
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
			if _, err := fmt.Fprintf(w, "%s %s\n", action.ActionType.StringShort(), key); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "Plan: %d to add, %d to change, %d to delete, %d unchanged\n", createCount, updateCount, deleteCount, unchangedCount)
	return err
}
