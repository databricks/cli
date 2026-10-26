package phases

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// recordedDeployment identifies the deployment a bind registers with, as read before the bind.
type recordedDeployment struct {
	deploymentID string
	lastVersion  int
}

// snapshotRecordedState writes the resources the service records to a plain local state file, so
// the bind can plan against them as it does against a state file. It closes the recorded state,
// which the caller opened for read. The deployment ID is empty when nothing is recorded yet.
func snapshotRecordedState(ctx context.Context, b *bundle.Bundle) (recordedDeployment, string, error) {
	db := &b.DeploymentBundle.StateDB
	recorded := recordedDeployment{deploymentID: db.DeploymentID, lastVersion: db.VersionID}
	data := dstate.NewDatabase(db.Data.Lineage, db.Data.Serial)
	maps.Copy(data.State, db.Data.State)
	if _, err := db.Finalize(ctx); err != nil {
		return recordedDeployment{}, "", err
	}

	_, localPath := b.StateFilenameDirect(ctx)
	snapshotPath := localPath + ".bind-snapshot"
	var snapshot dstate.DeploymentState
	snapshot.OpenWithData(snapshotPath, data)
	if err := snapshot.Persist(); err != nil {
		return recordedDeployment{}, "", err
	}
	_, err := snapshot.Finalize(ctx)
	return recorded, snapshotPath, err
}

// recordBind registers the bound resource with the service: in a new version it records one bind
// operation carrying the state the bind resolved, creating the deployment on a first bind.
func recordBind(ctx context.Context, b *bundle.Bundle, recorded recordedDeployment, resolvedPath, resourceKey string) {
	entry, err := readResolvedEntry(ctx, resolvedPath, resourceKey)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	// The operation buffer records each state write through the workspace client on the context.
	w := b.WorkspaceClient(ctx)
	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, w)
	}

	var current *bundledeployments.Deployment
	if recorded.deploymentID != "" {
		current, err = w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{Name: dms.DeploymentName(recorded.deploymentID)})
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	db := &b.DeploymentBundle.StateDB
	_, localPath := b.StateFilenameDirect(ctx)
	err = db.Open(ctx, localPath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: recorded.deploymentID, LastVersionID: recorded.lastVersion})
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	// Closes the state on an early return; a no-op once it is finalized below.
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
	}()

	createOrUpdateDeployment(ctx, b, current)
	if logdiag.HasError(ctx) {
		return
	}
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	staged := []bundledeployments.StagedOperation{{
		ResourceKey: strings.TrimPrefix(resourceKey, dms.StatePrefix),
		ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
	}}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if err := db.SaveState(ctx, resourceKey, entry.ID, entry.State, entry.DependsOn); err != nil {
		logdiag.LogError(ctx, err)
	}
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
	}
	// As in deploy, push the state even after a failure, then close the version out.
	statemgmt.PushResourcesState(ctx, b)
	if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
		logdiag.LogError(ctx, err)
	}
}

// readResolvedEntry reads the state the bind resolved for resourceKey from its temporary state file.
func readResolvedEntry(ctx context.Context, path, resourceKey string) (dstate.ResourceEntry, error) {
	var resolved dstate.DeploymentState
	if err := resolved.Open(ctx, path, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		return dstate.ResourceEntry{}, err
	}
	entry, ok := resolved.GetResourceEntry(resourceKey)
	if _, err := resolved.Finalize(ctx); err != nil {
		return dstate.ResourceEntry{}, err
	}
	if !ok {
		return dstate.ResourceEntry{}, fmt.Errorf("internal error: bind resolved no state for %s", resourceKey)
	}
	return entry, nil
}
