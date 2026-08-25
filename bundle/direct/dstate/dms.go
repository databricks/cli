package dstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dms"
)

// RecordedState is what the CLI serializes into the DMS operation's state field. It wraps the
// config so depends_on survives the round trip: DMS has no field for dependency edges, and
// nesting them in the config would collide with resource fields of the same name.
type RecordedState struct {
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

// readDMSState replaces the file-derived resource state with what DMS recorded. Recording is
// only enabled for net-new deployments, so DMS owns the resource set outright: an empty set
// means a successful deploy of nothing, not missing data. The caller holds db.mu.
func (db *DeploymentState) readDMSState(ctx context.Context, src *DMSSource) error {
	recorded, err := src.Client.ListResources(ctx, src.DeploymentID)
	if err != nil {
		return err
	}

	// Built first and assigned together, so a malformed envelope leaves the state as it was
	// rather than half replaced.
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
