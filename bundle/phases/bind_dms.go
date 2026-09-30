package phases

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deployplan"
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

	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, wsc)
	}
	db := &b.DeploymentBundle.StateDB
	_, localPath := b.StateFilenameDirect(ctx)
	if err := db.Open(ctx, localPath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: deploymentID, LastVersionID: lastVersionID}); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	defer completeRecordedVersion(ctx, b)

	if existingID := db.GetResourceID(resourceKey); existingID != "" {
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
		return
	}

	// Plan the resource as an adoption of the existing id, then narrow the plan to it (and its own
	// grants/permissions) so the bind leaves the rest of the deployment untouched.
	b.DeploymentBundle.BindKey = resourceKey
	b.DeploymentBundle.BindID = resourceID
	plan, err := b.DeploymentBundle.CalculatePlan(ctx, wsc, &b.Config)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if err := selectBindPlan(plan, resourceKey); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if !confirmBindPlan(ctx, resourceKey, plan, autoApprove, true) {
		return
	}

	// Commit now: claim a version, apply the adoption, and complete it.
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

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

	b.DeploymentBundle.Apply(ctx, wsc, plan, false)
}

// completeRecordedVersion drains the buffered operations and closes the version out, completing
// with failure if anything went wrong. Before a version is claimed, it only closes the state.
func completeRecordedVersion(ctx context.Context, b *bundle.Bundle) {
	db := &b.DeploymentBundle.StateDB
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
	}
	if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// selectBindPlan includes the bound resource and its children, keeping unchanged dependencies
// for reference resolution. A bind must not deploy changes to other resources.
func selectBindPlan(plan *deployplan.Plan, resourceKey string) error {
	plan.FilterToSelected([]string{strings.TrimPrefix(resourceKey, "resources.")})
	for _, action := range plan.GetActions() {
		if action.ResourceKey == resourceKey || strings.HasPrefix(action.ResourceKey, resourceKey+".") || action.ActionType == deployplan.Skip {
			continue
		}
		return fmt.Errorf("cannot bind %s: dependency %s requires %s; deploy or bind the dependency first", resourceKey, action.ResourceKey, action.ActionType)
	}
	return nil
}
