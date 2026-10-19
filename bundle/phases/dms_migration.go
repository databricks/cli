package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/dmsmigration"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
)

// MigrateToDMS commits deployment history before ordinary deployment planning begins.
func MigrateToDMS(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalDeploy))
	if logdiag.HasError(ctx) {
		return
	}
	defer bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy))

	db := &b.DeploymentBundle.StateDB
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
	}()

	// Another deploy may have completed since the command's initial state read.
	state := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true))
	if logdiag.HasError(ctx) {
		return
	}
	if state.IsDMS() {
		b.MigratingToDirect = false
		return
	}
	if b.MigratingToDirect {
		if !state.Engine.IsDirect() {
			statemgmt.CommitMigration(ctx, b)
			if logdiag.HasError(ctx) {
				return
			}
			state = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true))
			if logdiag.HasError(ctx) {
				return
			}
		}
		b.MigratingToDirect = false
	}
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if !state.Engine.IsDirect() {
		logdiag.LogError(ctx, fmt.Errorf("deployment state changed to %s while enabling deployment history; retry the deployment", state.Engine))
		return
	}
	_, localPath := b.StateFilenameDirect(ctx)
	if err := db.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	deploymentID, deployment, err := dms.ReadDeployment(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	lastVersion, err := dms.ParseLastVersionID(deployment)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if err := statemgmt.BackupRemoteResourcesState(ctx, b); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	db.DeploymentID = deploymentID
	db.VersionID = lastVersion
	createOrUpdateDeployment(ctx, b, deployment)
	if logdiag.HasError(ctx) {
		return
	}
	ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	cmdio.LogString(ctx, "Enabling deployment history...")
	if err := dmsmigration.Migrate(ctx, db); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	logDeploymentVersion(ctx, b, "Deployment history enabled")
	statemgmt.PushResourcesState(ctx, b)
}
