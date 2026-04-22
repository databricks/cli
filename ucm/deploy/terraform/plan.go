package terraform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// PlanResult is the outcome of a Plan call. PlanPath points at the on-disk
// plan artefact Apply will consume; HasChanges is false when the plan is
// empty (terraform exits 0 with no diffs) so callers can short-circuit.
type PlanResult struct {
	// PlanPath is the absolute on-disk path of the saved plan (-out).
	PlanPath string
	// HasChanges is true when terraform plan detected at least one change.
	HasChanges bool
	// Summary is a one-line human-readable summary ("plan has changes" /
	// "no changes"). Retained for callers/tests that only care about the
	// legacy one-liner.
	Summary string
	// Plan is the DAB-parity structured plan, populated by translating the
	// saved plan file. Empty when there are no changes.
	Plan *deployplan.Plan
}

// Plan runs `terraform plan -out=<planfile>` in the working directory after
// ensuring init has happened. The returned *PlanResult is kept on the
// *Terraform as well, so a later Apply can pick up the plan artefact
// without the caller having to thread it through.
func (t *Terraform) Plan(ctx context.Context, u *ucm.Ucm) (*PlanResult, error) {
	if t == nil {
		return nil, fmt.Errorf("terraform: nil wrapper")
	}

	if err := t.Init(ctx, u); err != nil {
		return nil, err
	}

	planPath := filepath.Join(t.WorkingDir, PlanFileName)
	hasChanges, err := t.runner.Plan(ctx, tfexec.Out(planPath))
	if err != nil {
		return nil, fmt.Errorf("terraform plan: %w", err)
	}

	t.lastPlanPath = planPath
	t.lastPlanExists = true

	tfPlan, err := t.runner.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, fmt.Errorf("terraform show plan: %w", err)
	}

	result := &PlanResult{
		PlanPath:   planPath,
		HasChanges: hasChanges,
		Summary:    planSummary(hasChanges),
		Plan:       translatePlan(ctx, tfPlan),
	}
	log.Infof(ctx, "terraform plan: %s (at %s)", result.Summary, filepath.ToSlash(planPath))
	return result, nil
}

func planSummary(hasChanges bool) string {
	if hasChanges {
		return "plan has changes"
	}
	return "no changes"
}
