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
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// ErrHistoryRequiresDirectState keeps Terraform conversion separate from history registration.
var ErrHistoryRequiresDirectState = errors.New("deployment history is not supported with Terraform state; first migrate to direct deployment with deployment history disabled")

// MigrateToFileHistory registers the open file snapshot before normal deployment planning.
// History errors only warn; resources.json remains the authoritative resource state.
func MigrateToFileHistory(ctx context.Context, b *bundle.Bundle, state *statemgmt.StateDesc) {
	if !state.Engine.IsDirect() {
		logdiag.LogError(ctx, ErrHistoryRequiresDirectState)
		return
	}
	db := &b.DeploymentBundle.StateDB
	recorded := state.RecordsHistory()
	if db.IsOpen() {
		recorded = db.RecordsHistory()
	}
	if recorded {
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalDeploy))
	if logdiag.HasError(ctx) {
		return
	}
	defer bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy))

	db.AssertOpenedForRead()
	// Another deployment may have finished while this command built artifacts.
	// Recheck under the lock before committing the snapshot loaded before the build.
	latest := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true))
	if logdiag.HasError(ctx) {
		return
	}
	if latest.Lineage != db.Data.Lineage || latest.Serial > db.Data.Serial {
		logdiag.LogError(ctx, errors.New("deployment state changed before enabling history; run 'bundle deploy' again"))
		return
	}
	w := b.WorkspaceClient(ctx)
	id, deployment, err := dms.ReadDeployment(ctx, w, b.Config.Workspace.StatePath)
	if err != nil {
		log.Warnf(ctx, "Could not initialize deployment history; continuing with file state: %s", err)
		return
	}
	lastVersion, err := dms.ParseLastVersionID(deployment)
	if err != nil {
		log.Warnf(ctx, "Could not initialize deployment history; continuing with file state: %s", err)
		return
	}
	if err := statemgmt.BackupRemoteResourcesState(ctx, b); err != nil {
		log.Warnf(ctx, "Could not back up resource state before history registration: %s", err)
	}
	db.DeploymentID, db.VersionID = id, lastVersion
	if err := createOrUpdateDeployment(ctx, b, deployment); err != nil {
		log.Warnf(ctx, "Could not initialize deployment history; continuing with file state: %s", err)
		return
	}
	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, w)
	}
	cmdio.LogString(ctx, "Enabling deployment history...")
	versionID, recordErr := registerMigrationState(ctx, db)
	if recordErr != nil {
		log.Warnf(ctx, "Initial history registration was incomplete; continuing with file state: %s", recordErr)
	}
	// Commit only the file marker. IDs, resource payloads and dependency edges stay intact.
	if err := db.MarkFileHistory(versionID); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("saving history-enabled file state: %w", err))
		return
	}
	logDeploymentVersion(ctx, b, "Deployment history enabled")
	statemgmt.PushResourcesState(ctx, b)
	state.Serial = db.Data.Serial
}

// registerMigrationState records every resource from the open direct state in an initial registration version.
// The caller must hold the deployment lock and have created the DMS deployment.
func registerMigrationState(ctx context.Context, db *dstate.DeploymentState) (int, error) {
	keys := slices.Sorted(maps.Keys(db.Data.State))
	if len(keys) == 0 {
		return 0, nil
	}
	states := make([]json.RawMessage, len(keys))
	operations := make([]bundledeployments.StagedOperation, len(keys))
	for i, key := range keys {
		entry := db.Data.State[key]
		state, err := json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
		if err != nil {
			return 0, fmt.Errorf("serializing migration state for %s: %w", key, err)
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
		return 0, fmt.Errorf("creating migration version: %w", err)
	}

	buffer := dms.StartOperationBuffer(ctx, db.DeploymentID, versionID, dms.BufferOptions{BestEffort: true, OperationCount: len(keys)})
	for i, key := range keys {
		buffer.RecordOperation(ctx, key, false, db.Data.State[key].ID, states[i])
	}
	recordErr := buffer.Drain(ctx)
	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if recordErr != nil {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}
	_, completeErr := w.BundleDeployments.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             dms.VersionName(db.DeploymentID, versionID),
		CompletionReason: reason,
	})
	if err := errors.Join(recordErr, completeErr); err != nil {
		return versionID, fmt.Errorf("recording migration version: %w", err)
	}

	return versionID, nil
}
