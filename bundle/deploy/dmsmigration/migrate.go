package dmsmigration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Migrate records every resource from the open resources.json state in a bind-only version.
// The caller must hold the deployment lock and have created the DMS deployment.
func Migrate(ctx context.Context, db *dstate.DeploymentState) error {
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
			ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
		}
	}

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

	return db.UseDeploymentHistory(versionID)
}
