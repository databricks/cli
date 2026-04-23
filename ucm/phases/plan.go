package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/deployplan"
)

// Plan runs the initialize → build → engine-specific plan sequence and
// returns a *PlanOutcome carrying the structured plan + summary bits.
// Errors are reported via logdiag; on error Plan returns nil and the
// caller should check logdiag.HasError before rendering any output.
//
// The terraform engine runs: initialize → build → tf init → tf plan.
// The direct engine runs: initialize → load state → compute diff.
//
// Plan does NOT call state.Push — a plan never advances remote state.
// The deploy-side lock is held only for the state.Pull in Initialize and
// released; planning itself runs lock-free because it never writes.
func Plan(ctx context.Context, u *ucm.Ucm, opts Options) *PlanOutcome {
	log.Info(ctx, "Phase: plan")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return nil
	}

	if setting.Type.IsDirect() {
		return planDirect(ctx, u, opts)
	}
	return planTerraform(ctx, u, opts)
}

func planTerraform(ctx context.Context, u *ucm.Ucm, opts Options) *PlanOutcome {
	ucm.ApplyContext(ctx, u, validate.ReferenceClosure())
	if logdiag.HasError(ctx) {
		return nil
	}

	tf := Build(ctx, u, opts)
	if tf == nil || logdiag.HasError(ctx) {
		return nil
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return nil
	}

	result, err := tf.Plan(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform plan: %w", err))
		return nil
	}
	if result == nil {
		return nil
	}

	return &PlanOutcome{
		Plan:       result.Plan,
		HasChanges: result.HasChanges,
		Summary:    result.Summary,
	}
}

func planDirect(ctx context.Context, u *ucm.Ucm, opts Options) *PlanOutcome {
	ucm.ApplyContext(ctx, u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	if logdiag.HasError(ctx) {
		return nil
	}
	ucm.ApplyContext(ctx, u, validate.ReferenceClosure())
	if logdiag.HasError(ctx) {
		return nil
	}

	state, err := direct.LoadState(direct.StatePath(u))
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return nil
	}

	plan := direct.CalculatePlan(u, state)
	hasChanges := planHasChanges(plan)
	_ = opts // direct engine doesn't currently need client access to plan
	return &PlanOutcome{
		Plan:       plan,
		HasChanges: hasChanges,
		Summary:    planSummary(hasChanges),
	}
}

// planHasChanges reports true when the plan contains at least one non-Skip
// action. Mirrors the terraform wrapper's HasChanges semantics so the
// two engines share a definition of "plan has changes".
func planHasChanges(plan *deployplan.Plan) bool {
	if plan == nil {
		return false
	}
	for _, entry := range plan.Plan {
		if entry.Action != deployplan.Skip && entry.Action != deployplan.Undefined {
			return true
		}
	}
	return false
}

// planSummary returns the legacy one-liner the terraform engine has always
// emitted. Kept local to this file because the terraform wrapper has its own
// identical helper; duplicating avoids a cross-package call for a 2-arm switch.
func planSummary(hasChanges bool) string {
	if hasChanges {
		return "plan has changes"
	}
	return "no changes"
}
