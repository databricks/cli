package phases

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// bindWithHistory records a bind for a deployment that tracks history with the metadata service.
// It reuses the engine's file-based bind against a throwaway copy of what the service holds to get
// the plan preview and resolved state, then records that state as a bind operation so the next
// deploy sees the resource as managed.
func bindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey, resourceID string, autoApprove bool) {
	wsc := b.WorkspaceClient(ctx)

	deploymentID, deployment, lastVersionID, err := dms.FetchDeployment(ctx, wsc, b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	seedPath, cleanup, err := seedStateFromService(ctx, b, deploymentID, lastVersionID)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	defer cleanup()

	result, err := b.DeploymentBundle.Bind(ctx, wsc, &b.Config, seedPath, resourceKey, resourceID)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !confirmBindPlan(ctx, resourceKey, result, autoApprove) {
		return
	}
	// The seed's temp state is discarded; the bind is recorded with the service instead.
	defer result.Cancel()

	// Read the resolved state from the throwaway state the engine wrote, not the state cache, which
	// the plan step overwrites (dropping the etag for dashboards/genie_spaces).
	entry, ok, err := resolvedEntry(ctx, result.TempStatePath, resourceKey)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !ok {
		logdiag.LogError(ctx, fmt.Errorf("internal error: no resolved state for %q after bind", resourceKey))
		return
	}

	recordBind(ctx, b, deploymentID, deployment, lastVersionID, resourceKey, entry)
}

// recordBind creates the deployment on a first bind, then records a single bind operation carrying
// the resolved state, so the deployment lists the resource as managed.
func recordBind(ctx context.Context, b *bundle.Bundle, deploymentID string, deployment *bundledeployments.Deployment, lastVersionID int, resourceKey string, entry dstate.ResourceEntry) {
	ctx = withWorkspaceClient(ctx, b)
	db := &b.DeploymentBundle.StateDB
	if err := openRecordedState(ctx, db, localStatePath(ctx, b), deploymentID, lastVersionID); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	// Creates the deployment on a first bind, or refreshes stale metadata.
	createOrUpdateDeployment(ctx, b, deployment)
	if logdiag.HasError(ctx) {
		return
	}

	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	staged := []dms.StagedOperation{{ResourceKey: resourceKey, ActionType: dms.ActionBind}}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// The version exists now, so close it out on every path; otherwise a failure recording the
	// operation leaks its lease, as deploy and destroy also guard against.
	defer completeRecordedVersion(ctx, b)

	if err := db.SaveState(ctx, resourceKey, entry.ID, entry.State, entry.DependsOn); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// unbindWithHistory records an unbind: the resource and its sub-resources (permissions, grants) are
// dropped from what the service records, so the next deploy re-creates them. The workspace
// resources themselves are left untouched.
func unbindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey string) {
	wsc := b.WorkspaceClient(ctx)

	deploymentID, _, lastVersionID, err := dms.FetchDeployment(ctx, wsc, b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if deploymentID == "" {
		// Nothing is recorded, so there is nothing to unbind.
		return
	}

	ctx = withWorkspaceClient(ctx, b)
	db := &b.DeploymentBundle.StateDB
	if err := openRecordedState(ctx, db, localStatePath(ctx, b), deploymentID, lastVersionID); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	keys := recordedKeys(db, resourceKey)
	if len(keys) == 0 {
		// The resource is not recorded, so unbind is a no-op, matching a file-based deployment.
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		return
	}

	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	staged := make([]dms.StagedOperation, 0, len(keys))
	for _, k := range keys {
		staged = append(staged, dms.StagedOperation{ResourceKey: k, ActionType: dms.ActionUnbind})
	}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// The version exists now, so close it out on every path (see recordBind).
	defer completeRecordedVersion(ctx, b)

	for _, k := range keys {
		if err := db.DeleteState(ctx, k, false); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		log.Infof(ctx, "Unbound %s", k)
	}
}

// completeRecordedVersion drains the buffered operations and closes the version out, completing
// with failure if anything went wrong. Deferred once a version exists so every path completes it.
func completeRecordedVersion(ctx context.Context, b *bundle.Bundle) {
	db := &b.DeploymentBundle.StateDB
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
	}
	if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// seedStateFromService writes a throwaway local state holding what the service currently records,
// so the engine's file-based bind can compute its plan and resolved state against it. The returned
// path does not exist yet when no deployment has been recorded, which the engine reads as empty
// state. cleanup removes the seed and any temporary files the engine leaves beside it.
func seedStateFromService(ctx context.Context, b *bundle.Bundle, deploymentID string, lastVersionID int) (string, func(), error) {
	seedPath := localStatePath(ctx, b) + ".bind-seed"
	cleanup := func() {
		for _, p := range []string{seedPath, seedPath + ".temp-bind", seedPath + ".wal"} {
			_ = os.Remove(p)
		}
	}

	if deploymentID == "" {
		return seedPath, cleanup, nil
	}

	ctx = withWorkspaceClient(ctx, b)
	var src dstate.DeploymentState
	if err := openRecordedState(ctx, &src, localStatePath(ctx, b), deploymentID, lastVersionID); err != nil {
		cleanup()
		return "", nil, err
	}
	err := src.SnapshotToPlainState(seedPath)
	if _, ferr := src.Finalize(ctx); ferr != nil {
		log.Warnf(ctx, "failed to finalize state: %v", ferr)
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return seedPath, cleanup, nil
}

// resolvedEntry reads back the state the engine's bind resolved for resourceKey from the throwaway
// state file it wrote, which is what a file-based bind would have persisted.
func resolvedEntry(ctx context.Context, tempStatePath, resourceKey string) (dstate.ResourceEntry, bool, error) {
	var src dstate.DeploymentState
	if err := src.Open(ctx, tempStatePath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		return dstate.ResourceEntry{}, false, err
	}
	entry, ok := src.GetResourceEntry(resourceKey)
	if _, err := src.Finalize(ctx); err != nil {
		log.Warnf(ctx, "failed to finalize state: %v", err)
	}
	return entry, ok, nil
}

// recordedKeys returns resourceKey and its sub-resource keys (permissions, grants, ...) that the
// service holds, sorted so the staged operations and their requests are deterministic.
func recordedKeys(db *dstate.DeploymentState, resourceKey string) []string {
	var keys []string
	for k := range db.Data.State {
		if k == resourceKey || strings.HasPrefix(k, resourceKey+".") {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// openRecordedState opens the deployment's recorded state for read, reading its resources from the
// metadata service.
func openRecordedState(ctx context.Context, db *dstate.DeploymentState, path, deploymentID string, lastVersionID int) error {
	return db.Open(ctx, path, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: deploymentID, LastVersionID: lastVersionID})
}

func localStatePath(ctx context.Context, b *bundle.Bundle) string {
	_, localPath := b.StateFilenameDirect(ctx)
	return localPath
}

func withWorkspaceClient(ctx context.Context, b *bundle.Bundle) context.Context {
	if !cmdctx.HasWorkspaceClient(ctx) {
		return cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	}
	return ctx
}
