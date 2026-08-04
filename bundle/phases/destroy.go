package phases

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
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
	{group: "pipelines", message: deletePipelineMessage},
	{group: "volumes", message: deleteVolumeMessage},
	{group: "database_instances", message: deleteDatabaseInstanceMessage},
	{group: "synced_database_tables", message: deleteSyncedDatabaseTableMessage},
	{group: "postgres_projects", message: deletePostgresProjectMessage},
	{group: "postgres_branches", message: deletePostgresBranchMessage},
	{group: "postgres_databases", message: deletePostgresDatabaseMessage},
	{group: "vector_search_indexes", message: deleteVectorSearchIndexMessage},
	{group: "genie_spaces", message: deleteGenieSpaceMessage},
}

func approvalForDestroy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	deleteActions := plan.GetActions()

	// Deletes of resources that are already gone remotely only clean up the state,
	// so they don't count as destructive actions and are not listed as deletions.
	deleteActions = slices.DeleteFunc(deleteActions, func(a deployplan.Action) bool { return a.Gone })

	err := checkForPreventDestroy(b, deleteActions)
	if err != nil {
		return false, err
	}

	if len(deleteActions) > 0 {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
			if a.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	logApprovalGroups(ctx, deleteActions, destroyApprovalGroups, true, deployplan.Delete)

	cmdio.LogString(ctx, "All files and directories at the following location will be deleted: "+b.Config.Workspace.RootPath)
	cmdio.LogString(ctx, "")

	if b.AutoApprove {
		return true, nil
	}

	return cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
}

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType, recorder *dms.Recorder) {
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

	// Complete the version before deleting the remote files. The deployment is a
	// node under the state directory, so files.Delete removes it and any later call
	// fails with 404. CompleteVersion is idempotent, so the deferred call in Destroy
	// is a no-op after this.
	if err := recorder.CompleteVersion(ctx, true); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, files.Delete())

	if !logdiag.HasError(ctx) {
		cmdio.LogString(ctx, "Destroy complete!")
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
		cmdio.LogString(ctx, "No active deployment found to destroy!")
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	// Set up DMS recording of this destroy as a version. The version is not
	// created until the destroy is approved (below), so a cancelled destroy
	// records nothing; the deferred CompleteVersion is a no-op until then. It is
	// deferred before lock.Release so it runs while the lock is still held.
	recorder, err := newDeploymentRecorder(ctx, b, engine, dms.VersionTypeDestroy)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	defer func() {
		if err := recorder.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
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

	hasApproval, err := approvalForDestroy(ctx, b, plan)
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
		if err := recorder.CreateVersion(ctx); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		if recorder != nil {
			b.DeploymentBundle.OpRec = direct.NewOperationRecorder(
				b.WorkspaceClient(ctx).BundleDeployments,
				recorder.DeploymentID(),
				recorder.Version(),
			)
		}
		destroyCore(ctx, b, plan, engine, recorder)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
