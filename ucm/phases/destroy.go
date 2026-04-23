package phases

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/deployplan"
)

func approvalForDestroy(ctx context.Context, _ *ucm.Ucm, plan *deployplan.Plan, opts Options) (bool, error) {
	if plan == nil {
		return true, nil
	}

	deleteActions := plan.GetActions()
	if len(deleteActions) == 0 {
		return true, nil
	}

	cmdio.LogString(ctx, "The following resources will be deleted:")
	for _, a := range deleteActions {
		cmdio.Log(ctx, a)
	}
	cmdio.LogString(ctx, "")

	catalogActions := filterGroup(deleteActions, "catalogs", deployplan.Delete)
	schemaActions := filterGroup(deleteActions, "schemas", deployplan.Delete)
	volumeActions := filterGroup(deleteActions, "volumes", deployplan.Delete)

	if len(catalogActions) > 0 {
		cmdio.LogString(ctx, deleteCatalogMessage)
		for _, a := range catalogActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(schemaActions) > 0 {
		cmdio.LogString(ctx, deleteSchemaMessage)
		for _, a := range schemaActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(volumeActions) > 0 {
		cmdio.LogString(ctx, deleteVolumeMessage)
		for _, a := range volumeActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if opts.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("the destroy requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
	}

	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}
	return approved, nil
}

// destroyPlanFromConfig builds a synthetic destroy plan marking every
// resource declared in u.Config.Resources for deletion. Used by
// destroyTerraform to surface the affected resources to approvalForDestroy
// without relying on a terraform-destroy plan artefact.
func destroyPlanFromConfig(u *ucm.Ucm) *deployplan.Plan {
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{}}
	add := func(group string, keys ...string) {
		for _, k := range keys {
			plan.Plan["resources."+group+"."+k] = &deployplan.PlanEntry{Action: deployplan.Delete}
		}
	}
	res := u.Config.Resources
	for k := range res.Catalogs {
		add("catalogs", k)
	}
	for k := range res.Schemas {
		add("schemas", k)
	}
	for k := range res.Volumes {
		add("volumes", k)
	}
	for k := range res.Grants {
		add("grants", k)
	}
	for k := range res.StorageCredentials {
		add("storage_credentials", k)
	}
	for k := range res.ExternalLocations {
		add("external_locations", k)
	}
	for k := range res.Connections {
		add("connections", k)
	}
	return plan
}

// destroyPlanFromState builds a synthetic destroy plan marking every
// resource recorded in the direct-engine state for deletion. Mirrors the
// plan direct.Destroy constructs internally; duplicating the walk here lets
// destroyDirect surface affected resources to approvalForDestroy before the
// actual delete calls fire.
func destroyPlanFromState(state *direct.State) *deployplan.Plan {
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{}}
	add := func(group string, keys ...string) {
		for _, k := range keys {
			plan.Plan["resources."+group+"."+k] = &deployplan.PlanEntry{Action: deployplan.Delete}
		}
	}
	for k := range state.Catalogs {
		add("catalogs", k)
	}
	for k := range state.Schemas {
		add("schemas", k)
	}
	for k := range state.Volumes {
		add("volumes", k)
	}
	for k := range state.Grants {
		add("grants", k)
	}
	for k := range state.StorageCredentials {
		add("storage_credentials", k)
	}
	for k := range state.ExternalLocations {
		add("external_locations", k)
	}
	for k := range state.Connections {
		add("connections", k)
	}
	return plan
}

// Destroy runs the initialize → terraform-init → terraform-destroy →
// state-push sequence for the terraform engine, or the direct engine's
// equivalent: delete every recorded resource and persist the emptied state.
//
// For the terraform engine, the Build phase is skipped — destroy operates
// on the already-rendered terraform config cached from the last apply
// (tf.Init re-renders from current ucm.yml which is still necessary for the
// resource graph). The final Push uploads the post-destroy terraform.tfstate
// so peers observe the emptied state.
//
// For the direct engine, destroy walks the recorded state in reverse UC
// dependency order (grants → schemas → catalogs) and issues per-resource
// delete calls. The state file is rewritten on every successful delete so a
// mid-destroy error leaves the file consistent with whatever actually
// survived the run.
func Destroy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: destroy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		destroyDirect(ctx, u, opts)
		return
	}
	destroyTerraform(ctx, u, opts)
}

func destroyTerraform(ctx context.Context, u *ucm.Ucm, opts Options) {
	ucm.ApplyContext(ctx, u, validate.ReferenceClosure())
	if logdiag.HasError(ctx) {
		return
	}

	factory := opts.terraformFactoryOrDefault()
	tf, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("build terraform wrapper: %w", err))
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	approved, err := approvalForDestroy(ctx, u, destroyPlanFromConfig(u), opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Destroy cancelled!")
		return
	}

	if err := tf.Destroy(ctx, u, opts.ForceLock); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform destroy: %w", err))
		return
	}

	pushBackend := opts.Backend
	pushBackend.ForceLock = opts.ForceLock
	if err := deploy.Push(ctx, u, pushBackend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}

func destroyDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
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

	approved, err := approvalForDestroy(ctx, u, destroyPlanFromState(state), opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Destroy cancelled!")
		return
	}

	_, destroyErr := direct.Destroy(ctx, u, client, state)
	if saveErr := direct.SaveState(statePath, state); saveErr != nil {
		if destroyErr == nil {
			logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", saveErr))
			return
		}
		log.Warnf(ctx, "save direct state after destroy error: %v", saveErr)
	}
	if destroyErr != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct destroy: %w", destroyErr))
	}
}
