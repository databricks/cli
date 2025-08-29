package terranova

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
)

func (b *BundleDeployer) Apply(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root) {
	if b.Graph == nil {
		panic("Planning is not done")
	}

	if len(b.Resources) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	b.StateDB.AssertOpened()

	b.Graph.Run(defaultParallelism, func(node deployplan.ResourceNode, failedDependency *deployplan.ResourceNode) bool {
		deployer, exists := b.Resources[node]
		if !exists {
			// Unexpected, this should be filtered at plan.
			return false
		}
		actionType := deployer.ActionType

		errorPrefix := fmt.Sprintf("cannot %s %s.%s", actionType.String(), node.Group, node.Key)

		// If a dependency failed, report and skip execution for this node by returning false
		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, failedDependency.String()))
			return false
		}

		settings, ok := SupportedResources[node.Group]
		if !ok {
			// Unexpected, this should be filtered at plan.
			return false
		}

		// The way plan currently works, is that it does not add resources with Noop action, turning them into Unset.
		// So we skip both, although at this point we will not see Noop here.
		if actionType == deployplan.ActionTypeUnset || actionType == deployplan.ActionTypeNoop {
			return true
		}

		d := Deployer{
			Group:      node.Group,
			Key:        node.Key,
			RemoteType: settings.RemoteType,
		}

		if actionType == deployplan.ActionTypeDelete {
			err := d.Destroy(ctx, client, &b.StateDB)
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

		err = d.Deploy(ctx, client, &b.StateDB, config, actionType)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		// After successful deployment, resolve any references that were delayed during planning
		// This includes ID references and remote state references
		if b.Graph.HasOutgoingEdges(node) {
			for _, reference := range b.Graph.OutgoingLabels(node) {
				value, err := deployer.ResolveReferenceRemote(ctx, &b.StateDB, reference)
				if err != nil {
					logdiag.LogError(ctx, fmt.Errorf("failed to resolve reference %q for %s.%s after deployment: %w", reference, node.Group, node.Key, err))
					return false
				}

				err = replaceReferenceWithValue(ctx, configRoot, reference, value)
				if err != nil {
					logdiag.LogError(ctx, fmt.Errorf("failed to replace reference %q with value %v for %s.%s: %w", reference, value, node.Group, node.Key, err))
					return false
				}
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
