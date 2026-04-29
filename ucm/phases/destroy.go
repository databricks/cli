package phases

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/direct"
	"github.com/databricks/cli/ucm/scripts"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func assertRootPathExists(ctx context.Context, u *ucm.Ucm) (bool, error) {
	w := u.WorkspaceClient()
	_, err := w.Workspace.GetStatusByPath(ctx, u.Config.Workspace.RootPath) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.

	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		log.Infof(ctx, "Root path does not exist: %s", u.Config.Workspace.RootPath)
		return false, nil
	}

	return true, err
}

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

// Destroy runs the initialize → terraform-init → terraform-destroy →
// state-push sequence for the terraform engine, or the direct engine's
// equivalent: load state, plan an all-delete pass via CalculatePlan(nil),
// approve, Apply, then Finalize the emptied state.
//
// For the terraform engine, the Build phase is skipped — destroy operates
// on the already-rendered terraform config cached from the last apply
// (tf.Init re-renders from current ucm.yml which is still necessary for the
// resource graph). The final Push uploads the post-destroy terraform.tfstate
// so peers observe the emptied state.
//
// For the direct engine, destroy uses the same DeploymentUcm machinery as
// Deploy with the configRoot argument set to nil — that signals
// CalculatePlan to walk only the recorded state, marking every entry for
// deletion. Apply then issues per-resource delete calls in dependency
// order; Finalize rewrites the (now-empty) state file even if a mid-apply
// failure stops short of full deletion.
func Destroy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: destroy")

	ok, err := assertRootPathExists(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !ok {
		cmdio.LogString(ctx, "No active deployment found to destroy!")
		return
	}

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPreDestroy))
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		destroyDirect(ctx, u, opts)
	} else {
		destroyTerraform(ctx, u, opts)
	}
	if logdiag.HasError(ctx) {
		return
	}

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPostDestroy))
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

	// Advance the local state cache before Push so the post-destroy remote
	// blob carries a fresh Seq/CliVersion/Timestamp/UUID. Push only mirrors
	// local.
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
}

// destroyDirect drives the all-delete pass through the same direct.DeploymentUcm
// machinery used by Deploy. Passing nil for the configRoot tells CalculatePlan
// to mark every entry recorded in state for deletion. Apply then issues the
// per-resource delete calls; Finalize persists the (post-destroy) state so a
// mid-destroy failure leaves the file consistent with whatever survived.
// Mirrors bundle.destroyCore's direct branch.
func destroyDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
	var d direct.DeploymentUcm
	if err := d.StateDB.Open(DirectStatePath(u)); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("open direct state: %w", err))
		return
	}

	// nil configRoot: every resource currently in state becomes a Delete.
	plan, err := d.CalculatePlan(ctx, u.WorkspaceClient(), nil)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct destroy plan: %w", err))
		return
	}

	approved, err := approvalForDestroy(ctx, u, plan, opts)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !approved {
		cmdio.LogString(ctx, "Destroy cancelled!")
		return
	}

	d.Apply(ctx, u.WorkspaceClient(), plan, direct.MigrateMode(false))
	// Always Finalize for non-empty plans — Apply may have partially deleted
	// resources before logging an error, so we must persist the surviving
	// entries before bubbling the error up. Skip for empty plans to avoid
	// creating a state file when nothing was destroyed.
	if len(plan.Plan) > 0 {
		if err := d.StateDB.Finalize(); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("finalize direct state: %w", err))
			return
		}
	}
}
