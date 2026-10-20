package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
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
	if err := registerMigrationState(ctx, db); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	logDeploymentVersion(ctx, b, "Deployment history enabled")
	statemgmt.PushResourcesState(ctx, b)
	// Remember the migration version and that state now lives in DMS.
	// The deployment code uses this flag to accept the history setting and read resources from DMS.
	state.Serial = db.GetSerial()
	state.Features = db.StateFeatures()
}

// registerMigrationState records every resource from the open direct state in an initial registration version.
// The caller must hold the deployment lock and have created the DMS deployment.
func registerMigrationState(ctx context.Context, db *dstate.DeploymentState) error {
	keys := slices.Sorted(maps.Keys(db.Data.State))
	states := make([]json.RawMessage, len(keys))
	operations := make([]bundledeployments.StagedOperation, len(keys))
	for i, key := range keys {
		entry := db.Data.State[key]
		state, err := json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
		if err != nil {
			return fmt.Errorf("serializing migration state for %s: %w", key, err)
		}
		states[i] = state
		operations[i] = bundledeployments.StagedOperation{
			ResourceKey: strings.TrimPrefix(key, dms.StatePrefix),
			ActionType:  bundledeployments.OperationActionTypeOperationActionTypeInitialRegister,
		}
	}

	// Migration uses max(state serial, last version)+1; deployment follows at the next version.
	versionID := max(db.Data.Serial, db.VersionID) + 1
	previousVersionID := ""
	if db.VersionID > 0 {
		previousVersionID = strconv.Itoa(db.VersionID)
	}
	w := cmdctx.WorkspaceClient(ctx)
	_, err := w.BundleDeployments.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    dms.DeploymentName(db.DeploymentID),
		VersionId: strconv.Itoa(versionID),
		Version: bundledeployments.Version{
			CliVersion:        build.GetInfo().Version,
			VersionType:       dms.VersionTypeDeploy,
			PreviousVersionId: previousVersionID,
			Operations:        operations,
		},
	})
	if err != nil {
		return fmt.Errorf("creating migration version: %w", err)
	}

	buffer := dms.StartOperationBuffer(ctx, db.DeploymentID, versionID)
	for i, key := range keys {
		buffer.RecordOperation(ctx, key, false, db.Data.State[key].ID, states[i])
	}
	recordErr := buffer.Drain()
	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if recordErr != nil {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}
	_, completeErr := w.BundleDeployments.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             dms.VersionName(db.DeploymentID, versionID),
		CompletionReason: reason,
	})
	if err := errors.Join(recordErr, completeErr); err != nil {
		return fmt.Errorf("recording migration version: %w", err)
	}

	return db.MarkDeploymentHistory(versionID)
}
