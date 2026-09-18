package phases

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
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

	// Deletes that only clean up the state (already gone remotely, or no delete
	// operation) are not destructive, so they are not listed as deletions and need no
	// approval. In particular this makes prevent_destroy inert for state-only
	// resources: nothing is destroyed.
	deleteActions = slices.DeleteFunc(deleteActions, func(a deployplan.Action) bool { return a.IsStateOnlyDelete() })

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

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType) {
	// Not reported per resource: destroy names them up front for consent and then
	// reports only a count, so there is no per-resource output to report into.
	b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan, false)

	// Flush WAL to local state file before deleting remote files.
	// Warn instead of hard-error: resources are already deleted, so proceed
	// with file cleanup regardless of whether state flush succeeds.
	if _, err := b.DeploymentBundle.StateDB.Finalize(ctx); err != nil {
		diags := diag.WarningFromErr(err)
		if len(diags) > 0 {
			logdiag.LogDiag(ctx, diags[0])
		}
	}

	if logdiag.HasError(ctx) {
		return
	}

	if b.DeploymentBundle.StateDB.IsDeploymentMetadataService() {
		// Complete version before deleting remote files; the deployment node is under statePath.
		completed, err := b.DeploymentBundle.StateDB.CompleteVersion(ctx, true)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		// A completed destroy's resources are gone, so its deployment record is deleted too.
		if completed {
			deploymentID := b.DeploymentBundle.StateDB.DeploymentID
			if err := b.DeploymentBundle.StateDB.DmsClient().DeleteDeployment(ctx, deploymentID); err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to delete deployment: %w", err))
				return
			}
		}
	}

	bundle.ApplyContext(ctx, b, files.Delete())

	// Print the summary even on error: resources that were deleted are still worth
	// reporting (an accurate partial count is a follow-up).
	if b.Quiet < bundle.QuietAll {
		// Count top-level resources only, matching the approval list above (which
		// skips children); this also keeps the count stable across engines. Gone
		// resources are excluded to match that list: they were already deleted
		// remotely, so applying their Delete only cleans up stale state and is not
		// a destruction to report.
		deleted := 0
		for _, a := range plan.GetActions() {
			if a.ActionType == deployplan.Delete && !a.IsChildResource() && !a.IsStateOnlyDelete() {
				deleted++
			}
		}
		cmdio.LogString(ctx, fmt.Sprintf("Destroy: %d deleted", deleted))
	}

	if logdiag.HasError(ctx) {
		return
	}

	// Remove the local state file now that the deployment is gone. Destroy only
	// deletes the remote state; leaving the local file behind keeps its lineage
	// around, so a later fresh deploy of the same bundle (e.g. from another
	// machine that has no local state) mints a new lineage that no longer matches
	// this lingering one, and every subsequent command fails with a lineage
	// mismatch. Destroy runs on a single engine, so remove only that engine's
	// state file. Log a removal failure but keep going; the destroy already
	// succeeded and its summary is printed above.
	_, localStatePath := b.StateFilenameDirect(ctx)
	if err := os.Remove(localStatePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		logdiag.LogError(ctx, err)
	}

	// Destroy leaves empty scaffolding directories behind once their contents are
	// gone (e.g. .internal/ and sync-snapshots/ after the state and sync files are
	// removed), so prune them rather than littering empty directories. Pruning stays
	// within the target's state dir, so a sibling engine's state (terraform/) or
	// another target is untouched. Cosmetic cleanup: a failure here does not fail the
	// destroy.
	stateDir := b.GetLocalStateDir(ctx)
	if _, err := removeEmptyDirs(stateDir, 0); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debugf(ctx, "cannot prune empty state directories under %s: %v", stateDir, err)
	}
}

// maxStateDirDepth caps removeEmptyDirs recursion. The local state tree is only a
// few levels deep (bundle/<target>/{.internal,sync-snapshots,terraform}); the cap is
// a defensive guard against a pathologically deep tree, not a limit ever hit in
// practice.
const maxStateDirDepth = 100

// removeEmptyDirs removes every empty directory in the subtree rooted at dir,
// bottom-up, and reports whether dir itself was removed. Symlinked entries count as
// content and are never traversed, so a symlinked provider mirror is left intact.
// depth is the caller's recursion depth; recursion stops with an error once it passes
// maxStateDirDepth.
func removeEmptyDirs(dir string, depth int) (bool, error) {
	if depth > maxStateDirDepth {
		return false, fmt.Errorf("state directory nesting exceeds %d levels at %s", maxStateDirDepth, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	empty := true
	for _, entry := range entries {
		if !entry.IsDir() {
			empty = false
			continue
		}
		removed, err := removeEmptyDirs(filepath.Join(dir, entry.Name()), depth+1)
		if err != nil {
			return false, err
		}
		if !removed {
			empty = false
		}
	}
	if !empty {
		return false, nil
	}
	if err := os.Remove(dir); err != nil {
		return false, err
	}
	return true, nil
}

// The destroy phase deletes artifacts and resources.
func Destroy(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) {
	log.Info(ctx, "Phase: destroy")

	if !engine.IsDirect() {
		logdiag.LogError(ctx, errors.New(TerraformStateRemovedMessage))
		return
	}

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

	// DMS recording of this destroy: the version is created after approval, so a cancelled
	// destroy records nothing. Deferred before lock.Release to hold the lock; a no-op once
	// destroyCore has completed the version.
	defer func() {
		if engine.IsDirect() && b.DeploymentBundle.StateDB.IsDeploymentMetadataService() {
			completed, err := b.DeploymentBundle.StateDB.CompleteVersion(ctx, !logdiag.HasError(ctx))
			if err != nil {
				logdiag.LogError(ctx, err)
			} else if completed {
				deploymentID := b.DeploymentBundle.StateDB.DeploymentID
				if err := b.DeploymentBundle.StateDB.DmsClient().DeleteDeployment(ctx, deploymentID); err != nil {
					logdiag.LogError(ctx, fmt.Errorf("failed to delete deployment: %w", err))
				}
			}
		}
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDestroy))
	}()

	plan, err := b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), nil)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
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
		// Start the version for this destroy, now that it is approved. Everything else recording
		// does follows from the buffer this opens.
		if engine.IsDirect() && b.DeploymentBundle.StateDB.IsDeploymentMetadataService() {
			staged, err := stagedOperations(plan)
			if err != nil {
				logdiag.LogError(ctx, err)
				return
			}
			if err := startVersion(ctx, b, dms.VersionTypeDestroy, staged); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
		}
		destroyCore(ctx, b, plan, engine)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
