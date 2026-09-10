package phases

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
)

// bindWithHistory adopts an existing workspace resource for a deployment that records history with
// the metadata service. The service is the source of truth, so the bind is planned and applied now
// (as a Bind or BindAndUpdate operation in its own version), unlike the file-based bind which
// defers the change to the next deploy.
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

	// Stamp deployment metadata into config (as a deploy does) so the recorded state matches and a
	// later plan sees no drift. A first bind's deployment_id is stamped after it is created below.
	firstBind := deploymentID == ""
	muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(lastVersionID + 1)}
	if !firstBind {
		muts = append(muts, metadata.AnnotateDeployment(deploymentID))
	}
	bundle.ApplySeqContext(ctx, b, muts...)
	if logdiag.HasError(ctx) {
		finalizeState(ctx, db)
		return
	}

	// Plan the resource as an adoption of the existing id, then narrow the plan to it (and its own
	// grants/permissions) so the bind leaves the rest of the deployment untouched.
	b.DeploymentBundle.BindKey = resourceKey
	b.DeploymentBundle.BindID = resourceID
	plan, err := b.DeploymentBundle.CalculatePlan(ctx, wsc, &b.Config)
	if err != nil {
		finalizeState(ctx, db)
		logdiag.LogError(ctx, err)
		return
	}
	plan.FilterToSelected([]string{strings.TrimPrefix(resourceKey, "resources.")})

	if !confirmBindPlan(ctx, resourceKey, plan, autoApprove, true) {
		finalizeState(ctx, db)
		return
	}

	// Commit now: claim a version, apply the adoption, and complete it.
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// From here the state is open for write and may hold a version; drain and complete it (with
	// failure on error) on every path, so nothing is left open or a version left dangling.
	defer completeRecordedVersion(ctx, b)

	if !createDeploymentAndStamp(ctx, b, deployment, firstBind) {
		return
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

	b.DeploymentBundle.Apply(ctx, wsc, plan)
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
