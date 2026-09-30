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

// bindWithHistory applies adoption immediately and records it in DMS, the deployment's
// authoritative state. File-based binds defer changes until the next deploy.
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

	// Stamp before planning to avoid reporting deployment metadata as drift.
	muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(lastVersionID + 1)}
	if deploymentID != "" {
		muts = append(muts, metadata.AnnotateDeployment(deploymentID))
	}
	bundle.ApplySeqContext(ctx, b, muts...)
	if logdiag.HasError(ctx) {
		return
	}

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

	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	createOrUpdateDeployment(ctx, b, deployment)
	if logdiag.HasError(ctx) {
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

// completeRecordedVersion closes state and completes any claimed version, including on failure.
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
