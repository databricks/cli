package phases

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
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
// The engine's bind computes the plan preview and resolved state against a throwaway copy of what
// the service holds, then the bind is recorded as an operation carrying that state, so the next
// deploy sees the resource as managed rather than new.
func bindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey, resourceID string, autoApprove bool) {
	wsc := b.WorkspaceClient(ctx)

	deploymentID, deployment, err := dms.FetchDeployment(ctx, wsc, b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	lastVersionID, err := deploymentVersion(deployment)
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

	// Record exactly what the engine resolved for the resource - the same state, id and
	// dependencies a file-based bind would persist (etags and all) - read straight back out of the
	// throwaway state it wrote rather than the state cache, which the plan step overwrites.
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
	if err := db.Open(ctx, localStatePath(ctx, b), dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: deploymentID, LastVersionID: lastVersionID}); err != nil {
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

	if err := db.SaveState(ctx, resourceKey, entry.ID, entry.State, entry.DependsOn); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if _, err := db.CompleteVersion(ctx, true); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// unbindWithHistory records an unbind: the resource and its sub-resources (permissions, grants) are
// dropped from what the service records, so the next deploy re-creates them. The workspace
// resources themselves are left untouched.
func unbindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey string) {
	wsc := b.WorkspaceClient(ctx)

	deploymentID, deployment, err := dms.FetchDeployment(ctx, wsc, b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if deploymentID == "" {
		// Nothing is recorded, so there is nothing to unbind.
		return
	}
	lastVersionID, err := deploymentVersion(deployment)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	ctx = withWorkspaceClient(ctx, b)
	db := &b.DeploymentBundle.StateDB
	if err := db.Open(ctx, localStatePath(ctx, b), dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: deploymentID, LastVersionID: lastVersionID}); err != nil {
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

	for _, k := range keys {
		if err := db.DeleteState(ctx, k, false); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		log.Infof(ctx, "Unbound %s", k)
	}

	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if _, err := db.CompleteVersion(ctx, true); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// seedStateFromService writes a throwaway local state holding what the service currently records,
// so the engine's file-based bind can compute its plan and resolved state against it. Returns a
// path that does not exist yet when no deployment has been recorded, which the engine reads as an
// empty state. cleanup removes the seed and any temporary files the engine leaves beside it.
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
	if err := src.Open(ctx, localStatePath(ctx, b), dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: deploymentID, LastVersionID: lastVersionID}); err != nil {
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

func deploymentVersion(deployment *bundledeployments.Deployment) (int, error) {
	if deployment == nil || deployment.LastVersionId == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(deployment.LastVersionId)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last_version_id %q: %w", deployment.LastVersionId, err)
	}
	return v, nil
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
