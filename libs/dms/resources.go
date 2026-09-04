package dms

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Resource is what DMS holds for one resource of a deployment, as of the last operation
// that recorded it.
type Resource struct {
	// Key is the bundle state key, as everything outside this package spells it.
	Key string
	ID  string

	// State is the state the last operation recorded, as the opaque string the service
	// stores, and empty when no operation recorded one.
	State string
}

// ListResources returns every resource DMS holds for the deployment.
func (c *Client) ListResources(ctx context.Context, deploymentID string) ([]Resource, error) {
	it := c.Service.ListResources(ctx, bundledeployments.ListResourcesRequest{
		Parent: DeploymentName(deploymentID),
	})

	var out []Resource
	for it.HasNext(ctx) {
		res, err := it.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing resources from the deployment history service: %w", err)
		}
		out = append(out, Resource{
			Key:   statePrefix + res.ResourceKey,
			ID:    res.ResourceId,
			State: res.State,
		})
	}
	return out, nil
}
