package phases

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
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

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType) error {
	var applyErr error
	if engine.IsDirect() {
		applyErr = b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan)
	} else {
		// Core destructive mutators for destroy. These require informed user consent.
		applyErr = bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Flush the apply error before the (potentially slow) state finalize below,
	// so the user sees the failure before that work runs.
	applyErr = logdiag.FlushError(ctx, applyErr)

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

	if applyErr != nil {
		return applyErr
	}

	if err := bundle.ApplyContext(ctx, b, files.Delete()); err != nil {
		return logdiag.FlushError(ctx, err)
	}

	cmdio.LogString(ctx, "Destroy complete!")
	return nil
}

// The destroy phase deletes artifacts and resources.
func Destroy(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) (err error) {
	log.Info(ctx, "Phase: destroy")

	ok, existErr := assertRootPathExists(ctx, b)
	if existErr != nil {
		return existErr
	}

	if !ok {
		cmdio.LogString(ctx, "No active deployment found to destroy!")
		return nil
	}

	if acquireErr := bundle.ApplyContext(ctx, b, lock.Acquire()); acquireErr != nil {
		return acquireErr
	}

	defer func() {
		// Flush the destroy error before releasing the lock so the user sees the
		// failure before this final API call runs.
		err = logdiag.FlushError(ctx, err)
		if releaseErr := bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDestroy)); releaseErr != nil && err == nil {
			err = logdiag.FlushError(ctx, releaseErr)
		}
	}()

	if !engine.IsDirect() {
		if seqErr := bundle.ApplySeqContext(ctx, b,
			// We need to resolve artifact variable (how we do it in build phase)
			// because some of the to-be-destroyed resource might use this variable.
			// Not resolving might lead to terraform "Reference to undeclared resource" error
			mutator.ResolveVariableReferencesWithoutResources("artifacts"),
			mutator.ResolveVariableReferencesOnlyResources("artifacts"),

			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("destroy")),
		); seqErr != nil {
			return seqErr
		}
	}

	var plan *deployplan.Plan
	if engine.IsDirect() {
		plan, err = b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), nil)
		if err != nil {
			return err
		}
	} else {
		tf := b.Terraform
		if tf == nil {
			return errors.New("terraform not initialized")
		}

		plan, err = terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
		if err != nil {
			return err
		}
	}

	hasApproval, approvalErr := approvalForDestroy(ctx, b, plan)
	if approvalErr != nil {
		return approvalErr
	}

	if !hasApproval {
		cmdio.LogString(ctx, "Destroy cancelled!")
		return nil
	}

	if engine.IsDirect() {
		// Upgrade from read (opened by process.go) to write mode
		if upgradeErr := b.DeploymentBundle.StateDB.UpgradeToWrite(); upgradeErr != nil {
			return upgradeErr
		}
	}

	return destroyCore(ctx, b, plan, engine)
}
