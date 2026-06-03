package statemgmt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/tmpdms"
)

// LoadStateFromDMS loads resource state from the deployment metadata service
// into the state DB. It builds the in-memory database from the server's
// resource list and opens the state DB with it, which is the read-mode
// equivalent of opening a local state file when DMS is not in use.
func LoadStateFromDMS(ctx context.Context, b *bundle.Bundle) error {
	if b.DeploymentID == "" {
		return nil
	}

	svc, err := tmpdms.NewDeploymentMetadataAPI(b.WorkspaceClient(ctx))
	if err != nil {
		return fmt.Errorf("failed to create metadata service client: %w", err)
	}

	resources, err := svc.ListResources(ctx, tmpdms.ListResourcesRequest{
		DeploymentID: b.DeploymentID,
	})
	if err != nil {
		return fmt.Errorf("failed to list resources from deployment metadata service: %w", err)
	}

	data := dstate.NewDatabase("", 0)
	for _, r := range resources {
		// The DMS stores keys without the "resources." prefix (e.g., "jobs.foo").
		// The state DB expects the full key (e.g., "resources.jobs.foo").
		resourceKey := "resources." + r.ResourceKey

		var stateBytes json.RawMessage
		if r.State != nil {
			stateBytes, err = json.Marshal(r.State)
			if err != nil {
				return fmt.Errorf("marshaling state for %s: %w", resourceKey, err)
			}
		}

		data.State[resourceKey] = dstate.ResourceEntry{
			ID:    r.ResourceID,
			State: stateBytes,
		}
	}

	// OpenWithData populates the resource-key→ID index that GetResourceID relies
	// on. Writing Data.State directly would leave that index empty, so deletes
	// would fail with "missing in state".
	_, localPath := b.StateFilenameDirect(ctx)
	b.DeploymentBundle.StateDB.OpenWithData(localPath, data)
	return nil
}
