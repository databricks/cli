package phases

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/direct"
	"github.com/databricks/cli/ucm/render"
)

// directStatePath returns the on-disk location of the direct-engine state
// file for u's currently-selected target. Mirrors bundle's
// StateFilenameDirect local-path: <RootPath>/.databricks/ucm/<target>/resources.json.
// The file is read by dstate.DeploymentState.Open and rewritten by Finalize.
func directStatePath(u *ucm.Ucm) string {
	return filepath.Join(u.RootPath, filepath.FromSlash(deploy.LocalCacheDir), u.Config.Ucm.Target, "resources.json")
}

// PreDeployChecks is common set of mutators between "ucm plan" and "ucm deploy".
// Note, it is not run in "ucm migrate" so it must not modify the config.
//
// validate.ReferenceClosure runs here so dangling ${resources.<kind>.<key>}
// typos surface as a clean diagnostic before either engine attempts work.
// Mirrors bundle's process_static_resources position; previously the check
// was duplicated inside planTerraform/deployTerraform and skipped entirely
// on the direct path, where the dynamic per-entry resolution inside
// direct.DeploymentUcm is not equivalent to the static closure check.
func PreDeployChecks(ctx context.Context, u *ucm.Ucm, e engine.EngineType) {
	ucm.ApplySeqContext(ctx, u,
		mutator.ValidateDirectOnlyResources(e),
		mutator.ValidateLifecycleStarted(e),
		validate.ReferenceClosure(),
	)
	if logdiag.HasError(ctx) {
		return
	}
	// Remote-drift detection is terraform-only; the direct engine has its own
	// drift phase (ucm/phases/drift.go). Empty kinds today keeps this a no-op
	// scaffold — concrete UC resource kinds get wired here in later tasks.
	if !e.IsDirect() {
		ucm.ApplyContext(ctx, u, terraform.CheckResourcesModifiedRemotely(nil))
	}
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

	PreDeployChecks(ctx, u, setting.Type)
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

// planDirect computes a plan via the ucm/direct.DeploymentUcm machinery:
// open the local dstate file, build the adapter set from the workspace
// client, then walk the config + state to produce the per-resource action
// list. Mirrors bundle.RunPlan's direct branch (b.DeploymentBundle.CalculatePlan).
//
// Plan never advances state — Finalize is reserved for Deploy/Destroy.
func planDirect(ctx context.Context, u *ucm.Ucm, _ Options) *PlanOutcome {
	var d direct.DeploymentUcm
	if err := d.StateDB.Open(directStatePath(u)); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("open direct state: %w", err))
		return nil
	}

	plan, err := d.CalculatePlan(ctx, u.WorkspaceClient(), &u.Config)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct plan: %w", err))
		return nil
	}

	hasChanges := planHasChanges(plan)
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
