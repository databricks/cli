package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/render"
)

// PreDeployChecks is common set of mutators between "ucm plan" and "ucm deploy".
// Note, it is not run in "ucm migrate" so it must not modify the config.
//
// When downgradeWarningToError is true, any warning emitted by the validator
// pack is promoted to an error. Mirrors bundle's PreDeployChecks contract:
// the plan path passes true (warnings during planning should fail) while the
// deploy path passes false (warnings during deploy are tolerated).
func PreDeployChecks(ctx context.Context, u *ucm.Ucm, downgradeWarningToError bool, e engine.EngineType) {
	ucm.ApplySeqContext(ctx, u,
		promoteWarningsIfRequested(mutator.ValidateDirectOnlyResources(e), downgradeWarningToError),
		promoteWarningsIfRequested(mutator.ValidateLifecycleStarted(e), downgradeWarningToError),
	)
	if logdiag.HasError(ctx) {
		return
	}
	// Remote-drift detection is terraform-only; the direct engine has its own
	// drift phase (ucm/phases/drift.go). Empty kinds today keeps this a no-op
	// scaffold — concrete UC resource kinds get wired here in later tasks.
	if !e.IsDirect() {
		ucm.ApplyContext(ctx, u, promoteWarningsIfRequested(terraform.CheckResourcesModifiedRemotely(nil), downgradeWarningToError))
	}
}

// promoteWarningsIfRequested returns m unchanged when promote is false. When
// promote is true it wraps m so warning-severity diagnostics are rewritten to
// errors before being forwarded. This is how PreDeployChecks honours the
// downgradeWarningToError contract without each validator needing a flag.
func promoteWarningsIfRequested(m ucm.Mutator, promote bool) ucm.Mutator {
	if !promote {
		return m
	}
	return &warningPromoter{wrapped: m}
}

type warningPromoter struct {
	wrapped ucm.Mutator
}

func (w *warningPromoter) Name() string {
	return w.wrapped.Name()
}

func (w *warningPromoter) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	diags := w.wrapped.Apply(ctx, u)
	for i := range diags {
		if diags[i].Severity == diag.Warning {
			diags[i].Severity = diag.Error
		}
	}
	return diags
}

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

	PreDeployChecks(ctx, u, true, setting.Type)
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
		Summary:    render.PlanSummary(hasChanges),
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
