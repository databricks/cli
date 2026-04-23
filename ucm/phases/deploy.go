package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/metadata"
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

	if setting.Type.IsDirect() {
		deployDirect(ctx, u, opts)
		return
	}
	deployTerraform(ctx, u, opts)
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

func deployDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
	ucm.ApplyContext(ctx, u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	if logdiag.HasError(ctx) {
		return
	}
	ucm.ApplyContext(ctx, u, validate.ReferenceClosure())
	if logdiag.HasError(ctx) {
		return
	}

	factory := opts.directClientFactoryOrDefault()
	client, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("resolve direct client: %w", err))
		return
	}

	statePath := direct.StatePath(u)
	state, err := direct.LoadState(statePath)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return
	}

	plan := direct.CalculatePlan(u, state)
	approved, err := approvalForDeploy(ctx, u, plan, opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Deployment cancelled!")
		return
	}
	applyErr := direct.Apply(ctx, u, client, plan, state)
	// Always persist state — Apply mutates it as it goes, so partial progress
	// from a mid-apply error must survive the process exit.
	if saveErr := direct.SaveState(statePath, state); saveErr != nil {
		if applyErr == nil {
			logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", saveErr))
			return
		}
		log.Warnf(ctx, "save direct state after apply error: %v", saveErr)
	}
	if applyErr != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct apply: %w", applyErr))
		return
	}

	uploadMetadataBestEffort(ctx, u, opts.Backend)
}
