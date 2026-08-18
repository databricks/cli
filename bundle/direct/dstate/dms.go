package dstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// ResourceKeyPrefix is what a state key carries and a DMS resource key does not: state calls
// a job "resources.jobs.foo", DMS calls it "jobs.foo". Both sides must use this one constant
// or operations land under keys nothing reads.
const ResourceKeyPrefix = "resources."

// RecordedState is what the CLI serializes into the DMS Operation.State field. It wraps the
// config so depends_on survives the round trip: DMS has no field for dependency edges, and
// nesting them in the config would collide with resource fields of the same name.
type RecordedState struct {
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

// OperationInfo is what a state write reports to DMS. The caller describes the write; the
// sink does not infer it from the state being nil.
type OperationInfo struct {
	// Action is the operation DMS records for this write.
	Action deployplan.ActionType

	// InProgress marks a write that is half of a larger change, so an interrupted deploy does
	// not leave the resource described as finished. Only a recreate's delete sets it; the
	// create that follows updates the same operation to succeeded.
	InProgress bool
}

// OperationSink records one resource operation with DMS. Every state write calls it, so what
// DMS holds mirrors the WAL. It returns no error: the upload runs in the background, and the
// deploy learns of a failure when the queue is drained.
type OperationSink interface {
	RecordOperation(ctx context.Context, resourceKey string, info OperationInfo, resourceID string, state json.RawMessage)
}

// readDMSState replaces the file-derived resource state with what DMS recorded. Recording is
// only enabled for net-new deployments, so DMS owns the resource set outright: an empty set
// means a successful deploy of nothing, not missing data. The caller holds db.mu.
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
		if res.State != "" {
			// The service stores state as an opaque string, so it arrives as the
			// serialized envelope the write side sent.
			if err := json.Unmarshal([]byte(res.State), &recorded); err != nil {
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
