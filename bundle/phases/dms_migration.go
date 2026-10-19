package phases

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/dmsmigration"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
)

// ErrDMSRequiresDirectState rejects enabling history before Terraform state is migrated to direct.
var ErrDMSRequiresDirectState = errors.New("deployment history is not supported with Terraform state; first migrate to direct deployment with deployment history disabled")

// MigrateToDMS commits the already-open direct state to DMS and updates its descriptor before deployment planning.
func MigrateToDMS(ctx context.Context, b *bundle.Bundle, state *statemgmt.StateDesc) {
	if !state.Engine.IsDirect() {
		logdiag.LogError(ctx, ErrDMSRequiresDirectState)
		return
	}
	if state.IsDMS() {
		b.MigratingToDirect = false
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalDeploy))
	if logdiag.HasError(ctx) {
		return
	}
	defer bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy))

	db := &b.DeploymentBundle.StateDB
	db.AssertOpenedForRead()
	defer db.Close()

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
	state.Serial = db.GetSerial()
	state.Lineage = db.Data.Lineage
	state.Features = db.StateFeatures()
	state.SourcePath = filepath.ToSlash(db.Path)
	state.IsLocal = true
	state.Content = nil
}
