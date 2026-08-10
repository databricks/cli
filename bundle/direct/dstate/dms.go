package dstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// ResourceKeyPrefix is what a state key carries and a DMS resource key does not:
// state calls a job "resources.jobs.foo", DMS calls it "jobs.foo". Stripped on the
// way out and re-added on the way back, so both sides must use this one constant or
// operations silently land under keys nothing reads.
const ResourceKeyPrefix = "resources."

// RecordedState is what the CLI serializes into the DMS Operation.State field. It
// wraps the config rather than being it, so depends_on survives the round trip: DMS
// has no field for dependency edges, and they cannot be recomputed once references
// are resolved to literals. Nesting them in the config would collide with resource
// fields of the same name (e.g. jobs.Task.depends_on).
type RecordedState struct {
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

// OperationSink records one resource operation with the deployment metadata service.
// SaveState and DeleteState call it for every state write, so what DMS holds mirrors
// the WAL - including the intermediate writes of a recreate.
//
// It does not return an error: the upload happens on a background worker, and the
// deploy learns about a failure when the queue is drained.
type OperationSink interface {
	RecordOperation(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state json.RawMessage)
}

// readDMSState replaces the file-derived resource state with the state recorded in
// DMS. Recording is only enabled for net-new deployments, so once a deployment
// exists DMS owns its resource set outright - an empty set means a successful deploy
// of nothing, not missing data. The caller holds db.mu.
func (db *DeploymentState) readDMSState(ctx context.Context, src *DMSSource) error {
	resources, err := fetchDeploymentResources(ctx, src.Client, src.DeploymentID)
	if err != nil {
		return err
	}

	db.Data.State = resources
	db.stateIDs = make(map[string]string, len(resources))
	for key, entry := range resources {
		db.stateIDs[key] = entry.ID
	}
	return nil
}

// fetchDeploymentResources lists every resource recorded for the deployment in
// DMS and maps them into state entries keyed by the fully-qualified resource key.
func fetchDeploymentResources(ctx context.Context, client bundledeployments.BundleDeploymentsInterface, deploymentID string) (map[string]ResourceEntry, error) {
	it := client.ListResources(ctx, bundledeployments.ListResourcesRequest{
		Parent: "deployments/" + deploymentID,
	})

	out := make(map[string]ResourceEntry)
	for it.HasNext(ctx) {
		res, err := it.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing resources from deployment metadata service: %w", err)
		}

		// DMS reports resource keys without the "resources." prefix (e.g.
		// "jobs.foo"), but the state DB keys are fully qualified
		// ("resources.jobs.foo"), so prepend it here.
		key := ResourceKeyPrefix + res.ResourceKey

		var recorded RecordedState
		if res.State != nil {
			// State is a string field, so it arrives as a quoted JSON string (see the
			// write side in direct.operationRecorder.upload). Unquote it, then parse
			// the envelope it holds.
			var envelope string
			if err := json.Unmarshal(*res.State, &envelope); err != nil {
				return nil, fmt.Errorf("interpreting state recorded for %s: %w", key, err)
			}
			if err := json.Unmarshal([]byte(envelope), &recorded); err != nil {
				return nil, fmt.Errorf("interpreting state recorded for %s: %w", key, err)
			}
		}

		out[key] = ResourceEntry{
			ID:        res.ResourceId,
			State:     recorded.State,
			DependsOn: recorded.DependsOn,
		}
	}
	return out, nil
}
