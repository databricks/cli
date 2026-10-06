package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

type migrationResource struct {
	key      string
	id       string
	recorded json.RawMessage
}

// MigrateToDMS copies an existing direct deployment into DMS and changes resources.json to the
// deployment-history marker only after every resource and the rollback backup are durable.
func MigrateToDMS(ctx context.Context, b *bundle.Bundle) {
	log.Info(ctx, "Phase: migrate to deployment history")

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalMigrate))
	if logdiag.HasError(ctx) {
		return
	}
	defer bundle.ApplyContext(ctx, b, lock.Release(lock.GoalMigrate))

	dmsClient, err := dms.NewClient(b.WorkspaceClient(ctx))
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	deploymentID, current, err := dms.LookupDeployment(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	deploymentID, err = ensureDeployment(ctx, b, deploymentID, current, dmsClient)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	recorded, err := dmsClient.ListResources(ctx, deploymentID)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	resources, staged, err := migrationResources(b.DeploymentBundle.StateDB.Data.State, recorded)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if len(staged) > 0 {
		if err := recordMigrationVersion(ctx, b, dmsClient, deploymentID, current, resources, staged); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	if err := statemgmt.BackupRemoteResourcesState(ctx, b); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if err := b.DeploymentBundle.StateDB.PersistDeploymentHistoryMarker(); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("persisting deployment history marker: %w", err))
		return
	}
	statemgmt.PushResourcesState(ctx, b)
	if logdiag.HasError(ctx) {
		// Keep the local cache on direct state when the remote commit did not complete. Its higher
		// serial wins over a possibly-written tombstone on the next retry.
		if err := b.DeploymentBundle.StateDB.Persist(); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("restoring local direct state after failed migration commit: %w", err))
		}
		return
	}

	cmdio.LogString(ctx, fmt.Sprintf("Migrate: %d resources moved to deployment history", len(b.DeploymentBundle.StateDB.Data.State)))
}

// migrationResources returns direct resources not already recorded with the same ID and state.
// A retry therefore resumes a partial migration instead of writing successful resources again.
func migrationResources(direct map[string]dstate.ResourceEntry, existing []dms.Resource) ([]migrationResource, []dms.StagedOperation, error) {
	recordedByKey := make(map[string]dms.Resource, len(existing))
	for _, resource := range existing {
		recordedByKey[resource.Key] = resource
	}

	resources := make([]migrationResource, 0, len(direct))
	staged := make([]dms.StagedOperation, 0, len(direct))
	for _, key := range slices.Sorted(maps.Keys(direct)) {
		entry := direct[key]
		state, err := json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
		if err != nil {
			return nil, nil, fmt.Errorf("serializing state for %s: %w", key, err)
		}
		if previous, ok := recordedByKey[key]; ok && previous.ID == entry.ID && previous.State == string(state) {
			continue
		}

		resources = append(resources, migrationResource{key: key, id: entry.ID, recorded: state})
		staged = append(staged, dms.StagedOperation{
			ResourceKey: key,
			ActionType:  dms.OperationActionTypeMigrate,
		})
	}
	return resources, staged, nil
}

// recordMigrationVersion writes one migration version and closes it as failed when any operation
// upload fails, leaving resources.json authoritative for a later retry.
func recordMigrationVersion(ctx context.Context, b *bundle.Bundle, dmsClient *dms.Client, deploymentID string, current *bundledeployments.Deployment, resources []migrationResource, staged []dms.StagedOperation) error {
	lastVersionID := ""
	if current != nil {
		lastVersionID = current.LastVersionId
	}
	versionID, err := migrationVersionID(b.DeploymentBundle.StateDB.GetSerial(), lastVersionID)
	if err != nil {
		return err
	}

	_, err = dmsClient.CreateVersion(ctx, deploymentID, versionID, dms.CreateVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       dms.VersionTypeDeploy,
		PreviousVersionId: lastVersionID,
		Operations:        staged,
		GitInfo:           versionGitInfo(b),
	})
	if err != nil {
		return fmt.Errorf("creating migration version: %w", err)
	}

	buffer := dms.StartOperationBuffer(ctx, dmsClient, deploymentID, versionID)
	for _, resource := range resources {
		buffer.RecordOperation(ctx, resource.key, false, resource.id, resource.recorded)
	}
	recordErr := buffer.Drain()

	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if recordErr != nil {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}
	completeErr := dmsClient.CompleteVersion(ctx, deploymentID, versionID, reason)
	if completeErr != nil {
		completeErr = fmt.Errorf("completing migration version: %w", completeErr)
	}
	if recordErr != nil {
		recordErr = fmt.Errorf("recording migration resources: %w", recordErr)
	}
	return errors.Join(recordErr, completeErr)
}

// migrationVersionID advances past both state stores so a retry never reuses a serial that either
// resources.json or DMS has already observed.
func migrationVersionID(directSerial int, lastDMSVersion string) (int, error) {
	nextDMSVersion, err := dms.NextVersion(lastDMSVersion)
	if err != nil {
		return 0, err
	}
	return max(directSerial+1, nextDMSVersion), nil
}
