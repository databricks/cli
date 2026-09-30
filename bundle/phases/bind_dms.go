package phases

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// bindWithHistory applies adoption immediately and records it in DMS, the deployment's
// authoritative state. The caller opens StateDB before entering this phase.
func bindWithHistory(ctx context.Context, b *bundle.Bundle, resourceKey, resourceID string, autoApprove bool) {
	wsc := b.WorkspaceClient(ctx)

	db := &b.DeploymentBundle.StateDB
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
			logdiag.LogError(ctx, err)
		}
	}()

	if existingID := db.GetResourceID(resourceKey); existingID != "" {
		logdiag.LogError(ctx, fmt.Errorf("%s is already bound to ID %q; rebinding is not supported with deployment history", resourceKey, existingID))
		return
	}

	// Stamp before planning to avoid reporting deployment metadata as drift.
	muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(db.VersionID + 1)}
	if db.DeploymentID != "" {
		muts = append(muts, metadata.AnnotateDeployment(db.DeploymentID))
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

	var deployment *bundledeployments.Deployment
	if db.DeploymentID != "" {
		deployment, err = db.DmsClient().Service.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
			Name: dms.DeploymentName(db.DeploymentID),
		})
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
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
