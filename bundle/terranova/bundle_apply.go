package terranova

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
)

func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root) {
	if b.Graph == nil {
		panic("Planning is not done")
	}

	if len(b.DeploymentUnits) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	b.StateDB.AssertOpened()

	b.Graph.Run(defaultParallelism, func(node deployplan.ResourceNode, failedDependency *deployplan.ResourceNode) bool {
		d, exists := b.DeploymentUnits[node]
		if !exists {
			// Unexpected, this should be filtered at plan.
			return false
		}
		errorPrefix := fmt.Sprintf("cannot %s %s.%s", d.ActionType.String(), node.Group, node.Key)

		// If a dependency failed, report and skip execution for this node by returning false
		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, failedDependency.String()))
			return false
		}

		// The way plan currently works, is that it does not add resources with Noop action, turning them into Unset.
		// So we skip both, although at this point we will not see Noop here.
		if d.ActionType == deployplan.ActionTypeUnset || d.ActionType == deployplan.ActionTypeNoop {
			return true
		}

		if d.ActionType == deployplan.ActionTypeDelete {
			err := d.Destroy(ctx, &b.StateDB)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			return true
		}

		// Fetch the references to ensure all are resolved
		myReferences, err := extractReferences(configRoot.Value(), node)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: reading references from config: %w", errorPrefix, err))
			return false
		}

		// At this point it's an error to have unresolved deps
		if len(myReferences) > 0 {
			// TODO: include the deps themselves in the message
			logdiag.LogError(ctx, fmt.Errorf("%s: unresolved deps", errorPrefix))
			return false
		}

		config, ok := configRoot.GetResourceConfig(node.Group, node.Key)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error when reading config", errorPrefix))
			return false
		}

		// TODO: redo calcDiff to downgrade planned action if possible (?)

		err = d.Deploy(ctx, &b.StateDB, config, d.ActionType)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		// Update resources.id after successful deploy so that future ${resources...id} refs are replaced
		if b.Graph.HasOutgoingEdges(node) {
			err = resolveIDReference(ctx, &b.StateDB, configRoot, node.Group, node.Key)
			if err != nil {
				// not using errorPrefix because resource was deployed
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", node.Group, node.Key, err))
				return false
			}
		}

		return true
	})

	// This must run even if deploy failed:
	err := b.StateDB.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}
