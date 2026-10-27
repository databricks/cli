package dstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// RecordedState is what the CLI serializes into the DMS operation's state field. It wraps the
// config so depends_on survives the round trip: DMS has no field for dependency edges, and
// nesting them in the config would collide with resource fields of the same name.
type RecordedState struct {
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

// LoadDMSState loads authoritative resources into an already-open read-only checkpoint.
// The service version must be at least as recent as the file's cached serial.
func (db *DeploymentState) LoadDMSState(ctx context.Context, deploymentID string, lastVersionID int) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.AssertOpenedForRead()

	if db.Data.Serial > lastVersionID {
		return fmt.Errorf("inconsistent deployment state: resource state serial (%d) is ahead of deployment history version (%d)", db.Data.Serial, lastVersionID)
	}
	if deploymentID != "" {
		w := cmdctx.WorkspaceClient(ctx)
		recorded, err := retry.With(ctx, retry.IsTransientRead, func() ([]bundledeployments.Resource, error) {
			return w.BundleDeployments.ListResourcesAll(ctx, bundledeployments.ListResourcesRequest{
				Parent: dms.DeploymentName(deploymentID),
			})
		})
		if err != nil {
			return fmt.Errorf("listing resources from the deployment history service: %w", err)
		}
		resources := make([]dms.Resource, len(recorded))
		for i, res := range recorded {
			// DMS keys drop the "resources." prefix; add it back for local state.
			resources[i] = dms.Resource{
				Key:   dms.StatePrefix + res.ResourceKey,
				ID:    res.ResourceId,
				State: res.State,
			}
		}
		if err := db.applyDMSState(resources); err != nil {
			return err
		}
	}
	db.DeploymentID, db.VersionID = deploymentID, lastVersionID
	return nil
}

// applyDMSState fills in the resource state from what DMS recorded. For a deployment that uses DMS
// resources.json does not track state. The service is queried for state for resources and then they
// are filled in-place in DeploymentState.Data.State:
func (db *DeploymentState) applyDMSState(recorded []dms.Resource) error {
	if len(db.Data.State) > 0 {
		return fmt.Errorf("internal error: state file for a recorded deployment carries %d resources, expected none", len(db.Data.State))
	}

	// Built first and assigned together, so a malformed envelope leaves the state alone rather than
	// half filled.
	resources := make(map[string]ResourceEntry, len(recorded))
	stateIDs := make(map[string]string, len(recorded))
	for _, res := range recorded {
		entry, err := stateEntry(res)
		if err != nil {
			return err
		}
		resources[res.Key] = entry
		stateIDs[res.Key] = entry.ID
	}

	db.Data.State = resources
	db.stateIDs = stateIDs
	return nil
}

// stateEntry unwraps the envelope the write path recorded for a resource.
func stateEntry(res dms.Resource) (ResourceEntry, error) {
	var recorded RecordedState
	if res.State != "" {
		// The service stores state as an opaque string, so it arrives as the serialized
		// envelope the write side sent.
		if err := json.Unmarshal([]byte(res.State), &recorded); err != nil {
			return ResourceEntry{}, fmt.Errorf("interpreting state recorded for %s: %w", res.Key, err)
		}
	}

	return ResourceEntry{
		ID:        res.ID,
		State:     recorded.State,
		DependsOn: recorded.DependsOn,
	}, nil
}
