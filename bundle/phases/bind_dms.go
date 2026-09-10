package phases

import (
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// bindWithHistory adopts an existing workspace resource for a deployment that tracks history with
// the metadata service. Unlike the file-based bind, which stages the change for the next deploy,
// DMS is the source of truth, so the bind is planned and applied here and now: the resource is
// recorded as a Bind (config already matches) or BindAndUpdate (config differs) operation in its
// own version.
func bindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey, resourceID string, autoApprove bool) {
	wsc := b.WorkspaceClient(ctx)

	deploymentID, deployment, lastVersionID, err := dms.FetchDeployment(ctx, wsc, b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	ctx = withWorkspaceClient(ctx, b)
	db := &b.DeploymentBundle.StateDB
	if err := openRecordedState(ctx, db, localStatePath(ctx, b), deploymentID, lastVersionID); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if existingID := db.GetResourceID(resourceKey); existingID != "" {
		finalizeState(ctx, db)
		logdiag.LogError(ctx, direct.ErrResourceAlreadyBound{ResourceKey: resourceKey, ExistingID: existingID, NewID: resourceID})
		return
	}

	// Stamp the deployment metadata into config so the adopted resource records the same state a
	// deploy would, and a later plan sees no drift. Mirrors the DMS setup in
	// cmd/bundle/utils.ProcessBundleRet; deployment_id is unknown until a first bind creates it, so
	// it is stamped into the plan afterwards (StampDeploymentIdForFirstVersion).
	firstBind := deploymentID == ""
	muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(lastVersionID + 1)}
	if !firstBind {
		muts = append(muts, metadata.AnnotateDeployment(deploymentID))
	}
	bundle.ApplySeqContext(ctx, b, muts...)
	if logdiag.HasError(ctx) {
		return
	}

	// Plan the resource as an adoption of the existing id, then keep only that resource so the bind
	// leaves the rest of the deployment untouched.
	b.DeploymentBundle.BindKey = resourceKey
	b.DeploymentBundle.BindID = resourceID
	plan, err := b.DeploymentBundle.CalculatePlan(ctx, wsc, &b.Config)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	scopeToResource(plan, resourceKey)

	if !confirmBindPlan(ctx, resourceKey, plan, autoApprove) {
		finalizeState(ctx, db)
		return
	}

	// Commit now: claim a version, apply the adoption, and complete it.
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	createOrUpdateDeployment(ctx, b, deployment)
	if logdiag.HasError(ctx) {
		return
	}
	if firstBind {
		if err := b.DeploymentBundle.StampDeploymentIdForFirstVersion(db.DeploymentID); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}
	staged, err := stagedOperations(plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// The version exists now, so complete it on every path (see completeRecordedVersion).
	defer completeRecordedVersion(ctx, b)

	b.DeploymentBundle.Apply(ctx, wsc, plan)
}

// scopeToResource sets every resource other than resourceKey to Skip, so an apply touches only that
// resource. The others stay in the plan so references from the adopted resource still resolve.
func scopeToResource(plan *deployplan.Plan, resourceKey string) {
	for key, entry := range plan.Plan {
		if key != resourceKey {
			entry.Action = deployplan.Skip
		}
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
		finalizeState(ctx, db)
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
	// The version exists now, so complete it on every path (see recordBind's completeRecordedVersion).
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

// finalizeState drains and closes the state without recording a version, for the paths that open it
// but do not commit (a no-op or a declined bind).
func finalizeState(ctx context.Context, db *dstate.DeploymentState) {
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
	}
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
