package phases

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func assertRootPathExists(ctx context.Context, b *bundle.Bundle) (bool, error) {
	w := b.WorkspaceClient(ctx)
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.RootPath)

	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.StatusCode == http.StatusNotFound {
		log.Infof(ctx, "Root path does not exist: %s", b.Config.Workspace.RootPath)
		return false, nil
	}

	return true, err
}

var destroyApprovalGroups = []approvalGroup{
	{group: "schemas", message: deleteSchemaMessage},
	// Pipelines are handled separately in approvalForDestroy so the message reflects each
	// pipeline's cascade_on_destroy setting; see logPipelineDeleteApproval.
	{group: "volumes", message: deleteVolumeMessage},
	{group: "database_instances", message: deleteDatabaseInstanceMessage},
	{group: "synced_database_tables", message: deleteSyncedDatabaseTableMessage},
	{group: "postgres_projects", message: deletePostgresProjectMessage},
	{group: "postgres_branches", message: deletePostgresBranchMessage},
	{group: "postgres_databases", message: deletePostgresDatabaseMessage},
	{group: "vector_search_indexes", message: deleteVectorSearchIndexMessage},
	{group: "genie_spaces", message: deleteGenieSpaceMessage},
}

// logPipelineDeleteApproval prints the pipeline deletions. If cascade_on_destroy is true, we will include
// a note that datasets will be deleted as well.
func logPipelineDeleteApproval(ctx context.Context, b *bundle.Bundle, actions []deployplan.Action, engine engine.EngineType, quiet bool) error {
	pipelineDeletes := filterGroup(actions, "pipelines", deployplan.Delete)

	var cascading, retaining []deployplan.Action
	for _, a := range pipelineDeletes {
		cascade, err := pipelineDeletionCascades(b, a, engine)
		if err != nil {
			return err
		}
		if cascade {
			cascading = append(cascading, a)
		} else {
			retaining = append(retaining, a)
		}
	}

	for _, grp := range []struct {
		message string
		actions []deployplan.Action
	}{
		{deletePipelineWithCascadeMessage, cascading},
		{deletePipelineNoCascadeMessage, retaining},
	} {
		if len(grp.actions) == 0 || quiet {
			continue
		}
		cmdio.LogString(ctx, grp.message)
		for _, a := range grp.actions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}
	return nil
}

func approvalForDestroy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType) (bool, error) {
	deleteActions := plan.GetActions()

	// Deletes of resources that are already gone remotely only clean up the state,
	// so they don't count as destructive actions and are not listed as deletions.
	deleteActions = slices.DeleteFunc(deleteActions, func(a deployplan.Action) bool { return a.Gone })

	err := checkForPreventDestroy(b, deleteActions)
	if err != nil {
		return false, err
	}

	// With --auto-approve there is no prompt, so this listing is informational and -qq
	// suppresses it. Without --auto-approve we are about to ask for consent and the user
	// must see what they are consenting to, so it prints at any -q level. The approval
	// helpers below still run either way: they also validate (e.g. pipeline cascade
	// lookups can fail), so skipping them would skip that.
	quiet := b.AutoApprove && b.Quiet >= bundle.QuietAll

	if len(deleteActions) > 0 && !quiet {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
			if a.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if !quiet {
		logApprovalGroups(ctx, deleteActions, destroyApprovalGroups, true, deployplan.Delete)
	}
	// Called even when quiet: the cascade lookup can fail, and that error must surface.
	if err := logPipelineDeleteApproval(ctx, b, deleteActions, engine, quiet); err != nil {
		return false, err
	}

	if !quiet {
		cmdio.LogString(ctx, "All files and directories at the following location will be deleted: "+b.Config.Workspace.RootPath)
		cmdio.LogString(ctx, "")
	}

	if b.AutoApprove {
		return true, nil
	}

	return cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
}

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType, recording dms.Recording) {
	if engine.IsDirect() {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan)
	} else {
		// Core destructive mutators for destroy. These require informed user consent.
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Flush WAL to local state file before deleting remote files.
	// Warn instead of hard-error: resources are already deleted, so proceed
	// with file cleanup regardless of whether state flush succeeds.
	if engine.IsDirect() {
		if _, err := b.DeploymentBundle.StateDB.Finalize(ctx); err != nil {
			diags := diag.WarningFromErr(err)
			if len(diags) > 0 {
				logdiag.LogDiag(ctx, diags[0])
			}
		}
	}

	if logdiag.HasError(ctx) {
		return
	}

	// Complete version before deleting remote files; the deployment node is under statePath.
	if err := recording.Finish(ctx, true); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, files.Delete())

	if !logdiag.HasError(ctx) && b.Quiet < bundle.QuietAll {
		// Count top-level resources only, matching the approval list above (which
		// skips children); this also keeps the count stable across engines. Gone
		// resources are included: they are excluded from the approval prompt because
		// they are not destructive, but destroying them does remove them from state,
		// and "bundle deploy" likewise counts them as deleted.
		deleted := 0
		for _, a := range plan.GetActions() {
			if a.ActionType == deployplan.Delete && !a.IsChildResource() {
				deleted++
			}
		}
		cmdio.LogString(ctx, fmt.Sprintf("Destroy: %d deleted", deleted))
	}
}

// The destroy phase deletes artifacts and resources.
func Destroy(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) {
	log.Info(ctx, "Phase: destroy")

	ok, err := assertRootPathExists(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if !ok {
		cmdio.LogProgress(ctx, "No active deployment found to destroy!")
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalDestroy))
	if logdiag.HasError(ctx) {
		return
	}

	// Set up DMS recording of this destroy. Version is created after approval; cancelled
	// destroy records nothing. Deferred before lock.Release to hold the lock.
	recording, err := newRecording(ctx, b, engine, dms.VersionTypeDestroy)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	defer func() {
		if err := recording.Finish(ctx, !logdiag.HasError(ctx)); err != nil {
			logdiag.LogError(ctx, err)
		}
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDestroy))
	}()

	if !engine.IsDirect() {
		bundle.ApplySeqContext(ctx, b,
			// We need to resolve artifact variable (how we do it in build phase)
			// because some of the to-be-destroyed resource might use this variable.
			// Not resolving might lead to terraform "Reference to undeclared resource" error
			mutator.ResolveVariableReferencesWithoutResources("artifacts"),
			mutator.ResolveVariableReferencesOnlyResources("artifacts"),

			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("destroy")),
		)
	}

	if logdiag.HasError(ctx) {
		return
	}

	var plan *deployplan.Plan
	if engine.IsDirect() {
		plan, err = b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), nil)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		tf := b.Terraform
		if tf == nil {
			logdiag.LogError(ctx, errors.New("terraform not initialized"))
			return
		}

		plan, err = terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	hasApproval, err := approvalForDestroy(ctx, b, plan, engine)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if hasApproval {
		if engine.IsDirect() {
			// Upgrade from read (opened by process.go) to write mode
			if err := b.DeploymentBundle.StateDB.UpgradeToWrite(); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
		}
		// Record the DMS version now that the destroy is approved and the state WAL
		// has been opened, then record each delete operation under it.
		staged, err := stagedOperations(plan)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		sink, err := recording.Start(ctx, staged)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		b.DeploymentBundle.OpRec = sink
		destroyCore(ctx, b, plan, engine, recording)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
