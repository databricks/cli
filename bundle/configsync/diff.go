package configsync

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/log"
)

// DetectChanges compares current remote state with the last deployed state
// and returns a map of resource changes.
func DetectChanges(ctx context.Context, b *bundle.Bundle) (map[string]deployplan.Changes, error) {
	changes := make(map[string]deployplan.Changes)

	deployBundle := &direct.DeploymentBundle{}
	// TODO: for Terraform engine we should read the state file, converted to direct state format, it should be created during deployment
	_, statePath := b.StateFilenameDirect(ctx)

	plan, err := deployBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate plan: %w", err)
	}

	for resourceKey, entry := range plan.Plan {
		resourceChanges := make(deployplan.Changes)

		if entry.Changes != nil {
			for path, changeDesc := range entry.Changes {
				if changeDesc.Remote != nil && changeDesc.Action != deployplan.Skip {
					resourceChanges[path] = changeDesc
				}
			}
		}

		if len(resourceChanges) != 0 {
			changes[resourceKey] = resourceChanges
		}

		log.Debugf(ctx, "Resource %s has %d changes", resourceKey, len(resourceChanges))
	}

	return changes, nil
}
