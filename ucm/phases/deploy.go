package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/direct"
	"github.com/databricks/cli/ucm/metadata"
	"github.com/databricks/cli/ucm/permissions"
	"github.com/databricks/cli/ucm/scripts"
)

func approvalForDeploy(ctx context.Context, _ *ucm.Ucm, plan *deployplan.Plan, opts Options) (bool, error) {
	if plan == nil {
		return true, nil
	}

	actions := plan.GetActions()
	types := []deployplan.ActionType{deployplan.Recreate, deployplan.Delete}
	catalogActions := filterGroup(actions, "catalogs", types...)
	schemaActions := filterGroup(actions, "schemas", types...)
	volumeActions := filterGroup(actions, "volumes", types...)

	if len(catalogActions) == 0 && len(schemaActions) == 0 && len(volumeActions) == 0 {
		return true, nil
	}

	if len(catalogActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateCatalogMessage)
		for _, a := range catalogActions {
			cmdio.Log(ctx, a)
		}
	}

	if len(schemaActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateSchemaMessage)
		for _, a := range schemaActions {
			cmdio.Log(ctx, a)
		}
	}

	if len(volumeActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateVolumeMessage)
		for _, a := range volumeActions {
			cmdio.Log(ctx, a)
		}
	}

	if opts.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
	}

	cmdio.LogString(ctx, "")
	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}
	return approved, nil
}

// Deploy runs the initialize → build → terraform-init → terraform-apply →
// state-push sequence for the terraform engine, or the direct-apply path for
// the direct engine. Errors are reported via logdiag; terraform-engine
// state.Push is only called when the apply succeeds, so a mid-apply failure
// leaves the remote state on the previous Seq and the local cache updated
// but un-acknowledged.
//
// The terraform apply acquires its own deploy lock for the write window; the
// preceding Pull (in Initialize) and the subsequent Push each acquire and
// release the lock independently. Between those two lock windows the lock is
// released — intentional, because holding a remote lock across a long
// terraform apply would create availability problems for other targets.
//
// The direct engine is lock-free at this layer: state is a per-root local
// file and the SDK calls it issues serialize naturally through the UC API.
// Cross-process contention on the same target is a known gap — follow-up.
func Deploy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: deploy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	// Precheck UC privileges before we attempt any mutating deploy work. Plan
	// is read-only and skips this; Deploy must bail here so users see a clean
	// permissions diagnostic instead of a mid-apply API error.
	for _, d := range permissions.PermissionDiagnostics(ctx, u) {
		logdiag.LogDiag(ctx, d)
	}
	if logdiag.HasError(ctx) {
		return
	}

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPreDeploy))
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		deployDirect(ctx, u, opts)
	} else {
		deployTerraform(ctx, u, opts)
	}
	if logdiag.HasError(ctx) {
		return
	}

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPostDeploy))
}

func deployTerraform(ctx context.Context, u *ucm.Ucm, opts Options) {
	ucm.ApplyContext(ctx, u, validate.ReferenceClosure())
	if logdiag.HasError(ctx) {
		return
	}

	tf := Build(ctx, u, opts)
	if tf == nil || logdiag.HasError(ctx) {
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	// Plan before Apply so approvalForDeploy can inspect the diff. Apply
	// consumes the saved plan artefact via tf.lastPlanPath and avoids
	// re-planning inline.
	var plan *deployplan.Plan
	if result, err := tf.Plan(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform plan: %w", err))
		return
	} else if result != nil {
		plan = result.Plan
	}

	approved, err := approvalForDeploy(ctx, u, plan, opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Deployment cancelled!")
		return
	}

	if err := tf.Apply(ctx, u, opts.ForceLock); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform apply: %w", err))
		return
	}

	// Advance the local state cache before Push so the on-remote record
	// carries a fresh Seq/CliVersion/Timestamp/UUID. Push only mirrors local.
	ucm.ApplyContext(ctx, u, deploy.StateUpdate())
	if logdiag.HasError(ctx) {
		return
	}

	pushBackend := opts.Backend
	pushBackend.ForceLock = opts.ForceLock
	if err := deploy.Push(ctx, u, pushBackend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}

	uploadMetadataBestEffort(ctx, u, opts.Backend)
}

// uploadMetadataBestEffort uploads provenance after a successful deploy. It
// mirrors DAB's post-apply provenance write. The deploy has already succeeded
// by the time this runs, so post-success failures degrade to a warning
// instead of being surfaced via logdiag — masking the success with a metadata
// glitch is the wrong tradeoff. Callers without a StateFiler (e.g. a
// direct-engine deploy that never configures a remote backend) get no-op
// semantics; this keeps both deploy paths able to call the helper unguarded.
func uploadMetadataBestEffort(ctx context.Context, u *ucm.Ucm, b deploy.Backend) {
	if b.StateFiler == nil {
		return
	}
	if err := metadata.Upload(ctx, u, b, metadata.Compute(ctx, u)); err != nil {
		log.Warnf(ctx, "ucm metadata: upload failed: %v", err)
	}
}

// deployDirect computes the plan via direct.DeploymentUcm.CalculatePlan,
// asks for approval if any destructive actions are present, then runs
// Apply. Whether Apply succeeds or not, Finalize is invoked for non-empty
// plans so partial progress (resources created before a mid-apply failure)
// survives the process exit. Mirrors bundle.deployCore's direct branch.
func deployDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
	var d direct.DeploymentUcm
	if err := d.StateDB.Open(directStatePath(u)); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("open direct state: %w", err))
		return
	}

	plan, err := d.CalculatePlan(ctx, u.WorkspaceClient(), &u.Config)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct plan: %w", err))
		return
	}

	approved, err := approvalForDeploy(ctx, u, plan, opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Deployment cancelled!")
		return
	}

	d.Apply(ctx, u.WorkspaceClient(), plan, direct.MigrateMode(false))
	// Always Finalize for non-empty plans — Apply may log errors via logdiag
	// while still having mutated state in memory; we must persist that
	// partial progress before bubbling the error up through HasError.
	// Skip Finalize for empty plans to avoid creating a state file when
	// nothing was deployed (mirrors bundle.deployCore).
	//
	// A Finalize error is logged but does NOT short-circuit metadata upload:
	// bundle.deployCore continues to subsequent steps after a Finalize log,
	// and the trailing logdiag.HasError check below is what gates the
	// process exit. Returning early here would silently skip metadata upload
	// on partial-progress failures.
	if len(plan.Plan) > 0 {
		if err := d.StateDB.Finalize(); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("finalize direct state: %w", err))
		}
	}
	if logdiag.HasError(ctx) {
		return
	}

	uploadMetadataBestEffort(ctx, u, opts.Backend)
}
